#!/usr/bin/bash

# function for cleaning up child processes
exit-ugreen-diskiomon() {
  kill $smart_check_pid 2>/dev/null
  kill $zpool_check_pid 2>/dev/null
  kill $disk_online_check_pid 2>/dev/null
  kill $standby_checker_pid 2>/dev/null
}

# trap exit and signals to ensure cleanup of child processes
trap 'exit-ugreen-diskiomon' EXIT
trap 'exit-ugreen-diskiomon' SIGTERM
trap 'exit-ugreen-diskiomon' SIGINT

# use variables from config file (unRAID specific)
if [[ -f /boot/config/plugins/ugreenleds-driver/settings.cfg ]]; then
  source /boot/config/plugins/ugreenleds-driver/settings.cfg
fi

# load environment variables
if [[ -f /etc/ugreen-leds.conf ]]; then
  source /etc/ugreen-leds.conf
fi

# led-disk mapping (see https://github.com/miskcoo/ugreen_dx4600_leds_controller/pull/4)
MAPPING_METHOD=${MAPPING_METHOD:=ata} # ata, hctl, serial
led_map=(disk1 disk2 disk3 disk4 disk5 disk6 disk7 disk8)

# hctl, $> lsblk -S -x hctl -o hctl,serial,name 
# NOTE: It is reported that the order below should be adjusted for each model. 
#       Please check the disk mapping section in https://github.com/miskcoo/ugreen_dx4600_leds_controller/blob/master/README.md.
hctl_map=("0:0:0:0" "1:0:0:0" "2:0:0:0" "3:0:0:0" "4:0:0:0" "5:0:0:0" "6:0:0:0" "7:0:0:0")
# serial number, $> lsblk -S -x hctl -o hctl,serial,name 
serial_map=(${DISK_SERIAL})
# ata number, $> ls /sys/block | egrep ata\d
ata_map=("ata1" "ata2" "ata3" "ata4" "ata5" "ata6" "ata7" "ata8")

if which dmidecode > /dev/null; then
    product_name=$(dmidecode --string system-product-name)
    case "${product_name}" in 
        DXP6800*)   # tested on DXP6800 Pro
            echo "Found UGREEN DXP6800 series" 
            hctl_map=("2:0:0:0" "3:0:0:0" "4:0:0:0" "5:0:0:0" "0:0:0:0" "1:0:0:0")
            ata_map=("ata3" "ata4" "ata5" "ata6" "ata1" "ata2")
            ;;
        DX4600*)   # tested on DX4600 Pro
            echo "Found UGREEN DX4600 series" 
            ;;
        DX4700*) 
            echo "Found UGREEN DX4700 series" 
            ;;
        DXP2800*)   # see issue #19
            echo "Found UGREEN DXP2800 series" 
            ;;
        DXP4800*) 
            echo "Found UGREEN DXP4800 series" 
            ;;
        DXP8800*)  # tested on DXP8800 Plus
            echo "Found UGREEN DXP8800 series" 
            # using the default mapping
            ;;
        *)
            if [[ "${MAPPING_METHOD}" == "hctl" || "${MAPPING_METHOD}" == "ata" ]]; then
                echo -e "\033[0;31mUsing the default HCTL order. Please check it maps to your disk slots correctly."
                echo -e "If you confirm that the HCTL order is correct, or find it is different, you can "
                echo -e "submit an issue to let us know, so we can update the script."
                echo -e "Please read the disk mapping section in the link below for more details. "
                echo -e "   https://github.com/miskcoo/ugreen_dx4600_leds_controller/blob/master/README.md\033[0m"
            fi
            ;;
    esac
elif [[ "${MAPPING_METHOD}" == "hctl" || "${MAPPING_METHOD}" == "ata" ]]; then
    echo -e "\033[0;31minstalling the tool `dmidecode` is suggested; otherwise the script cannot detect your device and adjust the hctl/ata_map\033[0m"
fi 
declare -A devices

# set monitor SMART information to true by default if not running unRAID
if [[ -f /etc/unraid-version ]]; then
    CHECK_SMART=false
else
    CHECK_SMART=${CHECK_SMART:=true}
fi
# polling rate for smartctl. 360 seconds by default
CHECK_SMART_INTERVAL=${CHECK_SMART_INTERVAL:=360}
# refresh interval from disk leds
LED_REFRESH_INTERVAL=${LED_REFRESH_INTERVAL:=0.1}

# whether to check zpool health
CHECK_ZPOOL=${CHECK_ZPOOL:=false}
# polling rate for checking zpool health. 5 seconds by default
CHECK_ZPOOL_INTERVAL=${CHECK_ZPOOL_INTERVAL:=5}
# enable debug logging for zpool checks
DEBUG_ZPOOL=${DEBUG_ZPOOL:=false}

# polling rate for checking disk online. 5 seconds by default
CHECK_DISK_ONLINE_INTERVAL=${CHECK_DISK_ONLINE_INTERVAL:=5}

COLOR_DISK_HEALTH=${COLOR_DISK_HEALTH:="255 255 255"}
COLOR_DISK_UNAVAIL=${COLOR_DISK_UNAVAIL:="255 0 0"}
COLOR_DISK_STANDBY=${COLOR_DISK_STANDBY:="0 0 255"}
COLOR_ZPOOL_FAIL=${COLOR_ZPOOL_FAIL:="255 0 0"}
COLOR_SMART_FAIL=${COLOR_SMART_FAIL:="255 0 0"}
BRIGHTNESS_DISK_LEDS=${BRIGHTNESS_DISK_LEDS:="255"}


{ lsmod | grep ledtrig_oneshot > /dev/null; } || { modprobe -v ledtrig_oneshot && sleep 2; }

function is_disk_healthy_or_standby() {
    if [[ "$1" == "$COLOR_DISK_HEALTH" || "$1" == "$COLOR_DISK_STANDBY" ]]; then
        return 0  # 0 means successful
    else
        return 1
    fi
}

function disk_enumerating_string() {
    if [[ $MAPPING_METHOD == ata ]]; then
        ls -ahl /sys/block | sed 's/\/$//' | awk '{
            if (match($0, /ata[0-9]+/)) {
                ata = substr($0, RSTART, RLENGTH);
                if (match($0, /[^\/]+$/)) {
                    basename = substr($0, RSTART, RLENGTH);
                    print basename, ata;
                }
            }
        }'
    elif [[ $MAPPING_METHOD == hctl || $MAPPING_METHOD == serial ]]; then
        lsblk -S -o name,${MAPPING_METHOD},tran | grep sata
    else
        echo Unsupported mapping method: ${MAPPING_METHOD}
        exit 1
    fi
}

echo Enumerating disks based on $MAPPING_METHOD...
declare -A dev_map
while read line
do
    blk_line=($line)
    key=${blk_line[1]}
    val=${blk_line[0]}
    dev_map[${key}]=${val}
    echo $MAPPING_METHOD ${key} ">>" ${dev_map[${key}]}
done <<< "$(disk_enumerating_string)"

# initialize LEDs
declare -A dev_to_led_map
for i in "${!led_map[@]}"; do
    led=${led_map[i]} 
    if [[ -d /sys/class/leds/$led ]]; then
        echo oneshot > /sys/class/leds/$led/trigger
        echo 1 > /sys/class/leds/$led/invert
        echo 100 > /sys/class/leds/$led/delay_on
        echo 100 > /sys/class/leds/$led/delay_off
        echo "$COLOR_DISK_HEALTH" > /sys/class/leds/$led/color
        echo "$BRIGHTNESS_DISK_LEDS" > /sys/class/leds/$led/brightness

        # find corresponding device
        _tmp_str=${MAPPING_METHOD}_map[@]
        _tmp_arr=(${!_tmp_str})
        
        if [[ -v "dev_map[${_tmp_arr[i]}]" ]]; then
            dev=${dev_map[${_tmp_arr[i]}]}

            if [[ -f /sys/class/block/${dev}/stat ]]; then
                devices[$led]=${dev}
                dev_to_led_map[$dev]=$led
            else
                # turn off the led if no disk installed on this slot
                echo 0 > /sys/class/leds/$led/brightness
                echo none > /sys/class/leds/$led/trigger
            fi
        else
            # turn off the led if no disk installed on this slot
            echo 0 > /sys/class/leds/$led/brightness
            echo none > /sys/class/leds/$led/trigger
        fi
    fi
done

# construct zpool device mapping
declare -A zpool_ledmap
if [ "$CHECK_ZPOOL" = true ]; then
    [ "$DEBUG_ZPOOL" = true ] && echo Enumerating zpool devices...
    while read line
    do
        zpool_dev_line=($line)
        zpool_dev_name=${zpool_dev_line[0]} 
        zpool_scsi_dev_name="unknown"
        # zpool_dev_state=${zpool_dev_line[1]}
        case "$zpool_dev_name" in
            sd*)
                # remove the trailing partition number
                zpool_scsi_dev_name=$(echo $zpool_dev_name | sed 's/[0-9]*$//')
                ;;
            dm*)
                # find the underlying block device of the encrypted device
                dm_slaves=($(ls /sys/block/${zpool_dev_name}/slaves))
                zpool_scsi_dev_name=${dm_slaves[0]}
                ;;
            *)
                echo Unsupported zpool device type ${zpool_dev_name}.
                ;;
        esac

        # if the detected scsi device can be found in the mapping array 
        #echo zpool $zpool_dev_name ">>" $zpool_scsi_dev_name ">>" ${dev_to_led_map[${zpool_scsi_dev_name}]}
        if [[ -v "dev_to_led_map[${zpool_scsi_dev_name}]" ]]; then
            # Store mapping with both full device name (with partition) and base name (without partition)
            # This ensures we can find the LED regardless of how zpool status reports the device
            zpool_ledmap[$zpool_dev_name]=${dev_to_led_map[${zpool_scsi_dev_name}]}
            # Also store with base name for devices that might be reported without partition number
            if [[ "$zpool_dev_name" != "$zpool_scsi_dev_name" ]]; then
                zpool_ledmap[$zpool_scsi_dev_name]=${dev_to_led_map[${zpool_scsi_dev_name}]}
            fi
            [ "$DEBUG_ZPOOL" = true ] && echo "zpool device" $zpool_dev_name ">>" $zpool_scsi_dev_name ">> LED:"${zpool_ledmap[$zpool_dev_name]}
        fi
    done <<< "$(zpool status -L | grep -E '^\s*(sd|dm)')"

    function zpool_check_loop() {
        # Track which devices have already been logged as faulted
        declare -A zpool_faulted_logged
        while true; do
            while read line
            do
                # Parse zpool status line more robustly
                # Format: "        sda     ONLINE       0     0     0"
                # We need to extract device name and state, handling variable whitespace
                zpool_dev_name=$(echo "$line" | awk '{print $1}')
                # Extract and trim whitespace from state using awk (no external dependencies)
                zpool_dev_state=$(echo "$line" | awk '{gsub(/^[ \t]+|[ \t]+$/, "", $2); print $2}')

                # Normalize device name to match the mapping key (which uses full name with partition)
                # The mapping was created with the full device name from zpool status, so we use it as-is
                # But we also need to check with the base device name (without partition) as fallback
                zpool_dev_lookup=${zpool_dev_name}
                zpool_dev_base=$(echo $zpool_dev_name | sed 's/[0-9]*$//')

                # Try lookup with full device name first, then base name in zpool_ledmap
                led=""
                if [[ -v "zpool_ledmap[${zpool_dev_lookup}]" ]]; then
                    led=${zpool_ledmap[$zpool_dev_lookup]}
                elif [[ -v "zpool_ledmap[${zpool_dev_base}]" ]]; then
                    led=${zpool_ledmap[$zpool_dev_base]}
                fi

                # Fallback: if not found in zpool_ledmap, try direct lookup in dev_to_led_map
                # This handles cases where the device wasn't mapped during initialization
                if [[ -z "$led" ]]; then
                    if [[ -v "dev_to_led_map[${zpool_dev_base}]" ]]; then
                        led=${dev_to_led_map[$zpool_dev_base]}
                    fi
                fi

                if [[ -n "$led" ]]; then
                    # Only flag actual failure states, not just "not ONLINE"
                    # Valid states: ONLINE, AVAIL, DEGRADED (during operations)
                    # Failure states: OFFLINE, FAULTED, UNAVAIL, REMOVED, CORRUPT
                    case "${zpool_dev_state}" in
                        OFFLINE|FAULTED|UNAVAIL|REMOVED|CORRUPT)
                            # This is a real failure - always set LED to red, overriding any other state
                            echo "$COLOR_ZPOOL_FAIL" > /sys/class/leds/$led/color
                            # Only log once per faulted device until it recovers
                            if [[ -z "${zpool_faulted_logged[$zpool_dev_name]}" ]]; then
                                if [ "$DEBUG_ZPOOL" = true ]; then
                                    echo "ZPOOL Disk failure detected on /dev/$zpool_dev_name (state: ${zpool_dev_state}) -> LED: $led at $(date +%Y-%m-%d' '%H:%M:%S)"
                                else
                                    echo "ZPOOL Disk failure detected on /dev/$zpool_dev_name (state: ${zpool_dev_state}) at $(date +%Y-%m-%d' '%H:%M:%S)"
                                fi
                                zpool_faulted_logged[$zpool_dev_name]=1
                            fi
                            ;;
                        ONLINE|AVAIL|DEGRADED|*)
                            # Device is healthy in zpool - reset LED to healthy if it was previously marked as zpool failure
                            led_color=$(cat /sys/class/leds/$led/color)
                            if [[ "$led_color" == "$COLOR_ZPOOL_FAIL" ]]; then
                                echo "$COLOR_DISK_HEALTH" > /sys/class/leds/$led/color
                                [ "$DEBUG_ZPOOL" = true ] && echo "ZPOOL Disk /dev/$zpool_dev_name recovered (state: ${zpool_dev_state}) at $(date +%Y-%m-%d' '%H:%M:%S)"
                            fi
                            # Clear the logged flag so we'll log again if it faults in the future
                            unset zpool_faulted_logged[$zpool_dev_name]
                            ;;
                    esac
                else
                    # Debug: log when we can't find LED mapping for a faulted device
                    case "${zpool_dev_state}" in
                        OFFLINE|FAULTED|UNAVAIL|REMOVED|CORRUPT)
                            [ "$DEBUG_ZPOOL" = true ] && echo "WARNING: ZPOOL device /dev/$zpool_dev_name (state: ${zpool_dev_state}) not found in LED mapping (looked for: ${zpool_dev_lookup}, ${zpool_dev_base})" >&2
                            ;;
                    esac
                fi
            done <<< "$(zpool status -L | grep -E '^\s*(sd|dm)')"

            sleep ${CHECK_ZPOOL_INTERVAL}s
        done
    }

    zpool_check_loop &
    zpool_check_pid=$!
fi

# check disk health if enabled
if [ "$CHECK_SMART" = true ]; then
(
    while true; do
        for led in "${!devices[@]}"; do

            led_color=$(cat /sys/class/leds/$led/color)
            if ! is_disk_healthy_or_standby "$led_color"; then
                continue;
            fi

            dev=${devices[$led]}

            # read the smart status return code, but ignore if the drive is on standby
            smartctl -H /dev/${dev} -n standby,0 &> /dev/null
            RET=$?

            # check return code for critical errors (any bit set except bit 5)
            # possible return bits set: https://invent.kde.org/system/kpmcore/-/merge_requests/28
            if (( $RET & ~32 )); then
                echo "$COLOR_SMART_FAIL" > /sys/class/leds/$led/color
                echo SMART Disk failure detected on /dev/$dev at $(date +%Y-%m-%d' '%H:%M:%S)
                continue
            fi
        done
        sleep ${CHECK_SMART_INTERVAL}s
    done
) &
smart_check_pid=$!
fi

# check disk online status
(
    while true; do
        for led in "${!devices[@]}"; do
            dev=${devices[$led]}

            led_color=$(cat /sys/class/leds/$led/color)
            if ! is_disk_healthy_or_standby "$led_color"; then
                continue;
            fi

            if [[ ! -f /sys/class/block/${dev}/stat ]]; then
                echo "$COLOR_DISK_UNAVAIL" > /sys/class/leds/$led/color 2>/dev/null
                echo Disk /dev/$dev went offline at $(date +%Y-%m-%d' '%H:%M:%S)
                continue
            fi
        done
        sleep ${CHECK_DISK_ONLINE_INTERVAL}s
    done
) &
disk_online_check_pid=$!


diskiomon_parameters() {
    for led in "${!devices[@]}"; do
        echo ${devices[$led]} $led
    done
}

# monitor disk standby modes
STANDBY_MON_PATH=${STANDBY_MON_PATH:=/usr/bin/ugreen-check-standby}
STANDBY_CHECK_INTERVAL=${STANDBY_CHECK_INTERVAL:=1}
if [ -f "${STANDBY_MON_PATH}" ]; then
    ${STANDBY_MON_PATH} ${STANDBY_CHECK_INTERVAL} "${COLOR_DISK_STANDBY}" "${COLOR_DISK_HEALTH}" $(diskiomon_parameters) &
    standby_checker_pid=$1
fi

# monitor disk activities
BLINK_MON_PATH=${BLINK_MON_PATH:=/usr/bin/ugreen-blink-disk}
if [ -f "${BLINK_MON_PATH}" ]; then

    ${BLINK_MON_PATH} ${LED_REFRESH_INTERVAL} $(diskiomon_parameters)

else 
    declare -A diskio_data_rw
    while true; do
        for led in "${!devices[@]}"; do

            # if $dev does not exist, diskio_new_rw="", which will be safe
            diskio_new_rw="$(cat /sys/block/${devices[$led]}/stat 2>/dev/null)"

            if [ "${diskio_data_rw[$led]}" != "${diskio_new_rw}" ]; then
                echo 1 > /sys/class/leds/$led/shot
            fi

            diskio_data_rw[$led]=$diskio_new_rw
        done

        sleep ${LED_REFRESH_INTERVAL}s

    done
fi

