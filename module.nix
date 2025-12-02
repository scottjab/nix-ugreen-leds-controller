packagePath:
{
  config,
  lib,
  pkgs,
  ...
}:

with lib;

let
  # Use the system's kernel packages, not the default pkgs kernel
  kernelPackages = config.boot.kernelPackages;

  # Always build the package with the system's kernel to ensure kernel module matches
  # Even if package is in pkgs, we rebuild it to ensure kernel module compatibility
  defaultPackage = pkgs.callPackage (import packagePath) {
    kernel = kernelPackages.kernel;
  };

  cfg = config.services.ugreen-leds;
  package = cfg.package;

  # Helper function to format RGB color as string
  formatColor = color: "${toString color.r} ${toString color.g} ${toString color.b}";

  # RGB color type
  rgbColor = types.submodule {
    options = {
      r = mkOption {
        type = types.int;
        description = "Red component (0-255)";
      };
      g = mkOption {
        type = types.int;
        description = "Green component (0-255)";
      };
      b = mkOption {
        type = types.int;
        description = "Blue component (0-255)";
      };
    };
  };
in
{
  options.services.ugreen-leds = {
    enable = mkEnableOption "UGREEN LEDs controller";

    package = mkOption {
      type = types.package;
      default = defaultPackage;
      defaultText = "pkgs.ugreen-leds-controller or built from source";
      description = "The ugreen-leds-controller package to use.";
    };

    kernelModule = {
      enable = mkEnableOption "Load the UGREEN LEDs kernel module";
    };

    probeLeds = {
      enable = mkEnableOption "Enable LED hardware probing service";
    };

    # Note: ugreen-power-led script doesn't exist in v0.3
    # powerLed = {
    #   enable = mkEnableOption "Enable power LED service";
    # };

    diskMonitor = {
      enable = mkEnableOption "Enable disk I/O monitoring service";

      mappingMethod = mkOption {
        type = types.enum [
          "ata"
          "hctl"
          "serial"
        ];
        default = "ata";
        description = "Method for mapping disks to LEDs (ata, hctl, or serial)";
      };

      checkSmart = mkOption {
        type = types.bool;
        default = true;
        description = "Check SMART information";
      };

      checkSmartInterval = mkOption {
        type = types.int;
        default = 360;
        description = "Polling rate for smartctl in seconds";
      };

      ledRefreshInterval = mkOption {
        type = types.float;
        default = 0.1;
        description = "Refresh interval for disk LEDs in seconds";
      };

      checkZpool = mkOption {
        type = types.bool;
        default = true;
        description = "Check ZFS pool health";
      };

      checkZpoolInterval = mkOption {
        type = types.int;
        default = 5;
        description = "Polling rate for checking ZFS pool health in seconds";
      };

      debugZpool = mkOption {
        type = types.bool;
        default = false;
        description = "Enable debug logging for ZFS pool checks";
      };

      checkDiskOnlineInterval = mkOption {
        type = types.int;
        default = 5;
        description = "Polling rate for checking disk online status in seconds";
      };

      colorDiskHealth = mkOption {
        type = rgbColor;
        default = {
          r = 255;
          g = 255;
          b = 255;
        };
        description = "Color for healthy disks (RGB)";
      };

      colorDiskUnavail = mkOption {
        type = rgbColor;
        default = {
          r = 255;
          g = 0;
          b = 0;
        };
        description = "Color for unavailable disks (RGB)";
      };

      colorDiskStandby = mkOption {
        type = rgbColor;
        default = {
          r = 0;
          g = 0;
          b = 255;
        };
        description = "Color for disks in standby (RGB)";
      };

      colorZpoolFail = mkOption {
        type = rgbColor;
        default = {
          r = 255;
          g = 0;
          b = 0;
        };
        description = "Color for failed ZFS pools (RGB)";
      };

      colorSmartFail = mkOption {
        type = rgbColor;
        default = {
          r = 255;
          g = 0;
          b = 0;
        };
        description = "Color for SMART failures (RGB)";
      };

      brightnessDiskLeds = mkOption {
        type = types.int;
        default = 255;
        description = "Brightness for disk LEDs (0-255)";
      };

      standbyMonPath = mkOption {
        type = types.str;
        default = "/usr/bin/ugreen-check-standby";
        description = "Path to standby monitoring binary";
      };

      standbyCheckInterval = mkOption {
        type = types.int;
        default = 1;
        description = "Standby check interval in seconds";
      };

      blinkMonPath = mkOption {
        type = types.str;
        default = "/usr/bin/ugreen-blink-disk";
        description = "Path to blink disk binary";
      };
    };

    networkMonitor = {
      enable = mkEnableOption "Enable network device monitoring service";

      interfaces = mkOption {
        type = types.listOf types.str;
        default = [ ];
        example = [ "enp2s0" ];
        description = "List of network interfaces to monitor";
      };

      colorNormal = mkOption {
        type = rgbColor;
        default = {
          r = 255;
          g = 255;
          b = 255;
        };
        description = "Normal network LED color (RGB)";
      };

      colorGatewayUnreachable = mkOption {
        type = rgbColor;
        default = {
          r = 255;
          g = 0;
          b = 0;
        };
        description = "Color when gateway is unreachable (RGB)";
      };

      colorLinkPurpleDefault = mkOption {
        type = rgbColor;
        default = {
          r = 128;
          g = 0;
          b = 128;
        };
        description = "Default purple color for 2000/5000 Mbps links (RGB)";
      };

      colorLink100 = mkOption {
        type = types.nullOr rgbColor;
        default = null;
        description = "Color for 100 Mbps links (RGB). If null, uses normal color.";
      };

      colorLink1000 = mkOption {
        type = types.nullOr rgbColor;
        default = null;
        description = "Color for 1000 Mbps links (RGB). If null, uses normal color.";
      };

      colorLink2000 = mkOption {
        type = types.nullOr rgbColor;
        default = null;
        description = "Color for 2000 Mbps links (RGB). If null, uses purple default.";
      };

      colorLink2500 = mkOption {
        type = types.nullOr rgbColor;
        default = null;
        description = "Color for 2500 Mbps links (RGB). If null, uses normal color.";
      };

      colorLink5000 = mkOption {
        type = types.nullOr rgbColor;
        default = null;
        description = "Color for 5000 Mbps links (RGB). If null, uses 10000 color or purple default.";
      };

      colorLink10000 = mkOption {
        type = types.nullOr rgbColor;
        default = null;
        description = "Color for 10000 Mbps links (RGB). If null, uses 5000 color or purple default.";
      };

      brightnessLed = mkOption {
        type = types.int;
        default = 255;
        description = "Brightness for network LED (0-255)";
      };

      checkInterval = mkOption {
        type = types.int;
        default = 60;
        description = "Network check interval in seconds";
      };

      checkGatewayConnectivity = mkOption {
        type = types.bool;
        default = false;
        description = "Check gateway connectivity";
      };

      checkLinkSpeed = mkOption {
        type = types.bool;
        default = false;
        description = "Check link speed and set color accordingly";
      };

      checkLinkSpeedDynamic = mkOption {
        type = types.bool;
        default = false;
        description = "Use dynamic color based on link speed";
      };

      checkLinkSpeedDynamicColorLow = mkOption {
        type = rgbColor;
        default = {
          r = 255;
          g = 0;
          b = 0;
        };
        description = "Color for low link speed in dynamic mode (RGB)";
      };

      checkLinkSpeedDynamicColorHigh = mkOption {
        type = rgbColor;
        default = {
          r = 0;
          g = 255;
          b = 0;
        };
        description = "Color for high link speed in dynamic mode (RGB)";
      };

      checkLinkSpeedDynamicSpeedLow = mkOption {
        type = types.int;
        default = 0;
        description = "Low speed threshold for dynamic color mode (Mbps)";
      };

      checkLinkSpeedDynamicSpeedHigh = mkOption {
        type = types.int;
        default = 10000;
        description = "High speed threshold for dynamic color mode (Mbps)";
      };

      blinkTx = mkOption {
        type = types.int;
        default = 1;
        description = "Enable TX blink (0 or 1)";
      };

      blinkRx = mkOption {
        type = types.int;
        default = 1;
        description = "Enable RX blink (0 or 1)";
      };

      blinkInterval = mkOption {
        type = types.int;
        default = 200;
        description = "Blink interval in milliseconds";
      };
    };
  };

  config = mkIf cfg.enable (
    let
      # Generate config file content
      configFileContent = ''
        # Disk Monitor Configuration
        MAPPING_METHOD=${cfg.diskMonitor.mappingMethod}
        CHECK_SMART=${if cfg.diskMonitor.checkSmart then "true" else "false"}
        CHECK_SMART_INTERVAL=${toString cfg.diskMonitor.checkSmartInterval}
        LED_REFRESH_INTERVAL=${toString cfg.diskMonitor.ledRefreshInterval}
        CHECK_ZPOOL=${if cfg.diskMonitor.checkZpool then "true" else "false"}
        CHECK_ZPOOL_INTERVAL=${toString cfg.diskMonitor.checkZpoolInterval}
        DEBUG_ZPOOL=${if cfg.diskMonitor.debugZpool then "true" else "false"}
        CHECK_DISK_ONLINE_INTERVAL=${toString cfg.diskMonitor.checkDiskOnlineInterval}
        COLOR_DISK_HEALTH="${formatColor cfg.diskMonitor.colorDiskHealth}"
        COLOR_DISK_UNAVAIL="${formatColor cfg.diskMonitor.colorDiskUnavail}"
        COLOR_DISK_STANDBY="${formatColor cfg.diskMonitor.colorDiskStandby}"
        COLOR_ZPOOL_FAIL="${formatColor cfg.diskMonitor.colorZpoolFail}"
        COLOR_SMART_FAIL="${formatColor cfg.diskMonitor.colorSmartFail}"
        BRIGHTNESS_DISK_LEDS=${toString cfg.diskMonitor.brightnessDiskLeds}
        STANDBY_MON_PATH=${cfg.diskMonitor.standbyMonPath}
        STANDBY_CHECK_INTERVAL=${toString cfg.diskMonitor.standbyCheckInterval}
        BLINK_MON_PATH=${cfg.diskMonitor.blinkMonPath}

        # Network Monitor Configuration
        COLOR_NETDEV_NORMAL="${formatColor cfg.networkMonitor.colorNormal}"
        COLOR_NETDEV_GATEWAY_UNREACHABLE="${formatColor cfg.networkMonitor.colorGatewayUnreachable}"
        COLOR_NETDEV_LINK_PURPLE_DEFAULT="${formatColor cfg.networkMonitor.colorLinkPurpleDefault}"
        ${optionalString (
          cfg.networkMonitor.colorLink100 != null
        ) ''COLOR_NETDEV_LINK_100="${formatColor cfg.networkMonitor.colorLink100}"''}
        ${optionalString (
          cfg.networkMonitor.colorLink1000 != null
        ) ''COLOR_NETDEV_LINK_1000="${formatColor cfg.networkMonitor.colorLink1000}"''}
        ${optionalString (
          cfg.networkMonitor.colorLink2000 != null
        ) ''COLOR_NETDEV_LINK_2000="${formatColor cfg.networkMonitor.colorLink2000}"''}
        ${optionalString (
          cfg.networkMonitor.colorLink2500 != null
        ) ''COLOR_NETDEV_LINK_2500="${formatColor cfg.networkMonitor.colorLink2500}"''}
        ${optionalString (
          cfg.networkMonitor.colorLink5000 != null
        ) ''COLOR_NETDEV_LINK_5000="${formatColor cfg.networkMonitor.colorLink5000}"''}
        ${optionalString (
          cfg.networkMonitor.colorLink10000 != null
        ) ''COLOR_NETDEV_LINK_10000="${formatColor cfg.networkMonitor.colorLink10000}"''}
        BRIGHTNESS_NETDEV_LED=${toString cfg.networkMonitor.brightnessLed}
        CHECK_NETDEV_INTERVAL=${toString cfg.networkMonitor.checkInterval}
        CHECK_GATEWAY_CONNECTIVITY=${
          if cfg.networkMonitor.checkGatewayConnectivity then "true" else "false"
        }
        CHECK_LINK_SPEED=${if cfg.networkMonitor.checkLinkSpeed then "true" else "false"}
        CHECK_LINK_SPEED_DYNAMIC=${if cfg.networkMonitor.checkLinkSpeedDynamic then "true" else "false"}
        CHECK_LINK_SPEED_DYNAMIC_COLOR_LOW="${formatColor cfg.networkMonitor.checkLinkSpeedDynamicColorLow}"
        CHECK_LINK_SPEED_DYNAMIC_COLOR_HIGH="${formatColor cfg.networkMonitor.checkLinkSpeedDynamicColorHigh}"
        CHECK_LINK_SPEED_DYNAMIC_SPEED_LOW=${toString cfg.networkMonitor.checkLinkSpeedDynamicSpeedLow}
        CHECK_LINK_SPEED_DYNAMIC_SPEED_HIGH=${toString cfg.networkMonitor.checkLinkSpeedDynamicSpeedHigh}
        NETDEV_BLINK_TX=${toString cfg.networkMonitor.blinkTx}
        NETDEV_BLINK_RX=${toString cfg.networkMonitor.blinkRx}
        NETDEV_BLINK_INTERVAL=${toString cfg.networkMonitor.blinkInterval}
      '';
    in
    {
      boot.kernelModules = mkIf cfg.kernelModule.enable [
        "i2c-dev"
        "led-ugreen"
        "ledtrig-oneshot"
        "ledtrig-netdev"
      ];

      boot.extraModulePackages = mkIf cfg.kernelModule.enable [
        package.kernelModule
      ];

      environment.systemPackages = [
        package
        pkgs.smartmontools # For smartctl command used by ugreen-diskiomon
        pkgs.iproute2 # For ping and ip commands used by ugreen-netdevmon
        pkgs.bc # For bc calculator used by ugreen-netdevmon
      ];

      systemd.services = mkMerge [
        (mkIf cfg.probeLeds.enable {
          ugreen-probe-leds = {
            description = "UGREEN LED initial hardware probing service";
            after = [ "systemd-modules-load.service" ];
            requires = [ "systemd-modules-load.service" ];
            serviceConfig = {
              Type = "oneshot";
              ExecStart = "${package}/bin/ugreen-probe-leds";
              RemainAfterExit = true;
              StandardOutput = "journal";
            };
            wantedBy = [ "multi-user.target" ];
          };
        })

        # Note: ugreen-power-led script doesn't exist in v0.3
        # (mkIf cfg.powerLed.enable {
        #   ugreen-power-led = {
        #     description = "UGREEN LEDs daemon for configuring power LED";
        #     after = [ "ugreen-probe-leds.service" ];
        #     requires = [ "ugreen-probe-leds.service" ];
        #     serviceConfig = {
        #       Type = "oneshot";
        #       ExecStart = "${package}/bin/ugreen-power-led";
        #       RemainAfterExit = true;
        #       StandardOutput = "journal";
        #     };
        #     wantedBy = [ "multi-user.target" ];
        #   };
        # })

        (mkIf cfg.diskMonitor.enable {
          ugreen-diskiomon = {
            description = "UGREEN LEDs daemon for monitoring diskio and blinking corresponding LEDs";
            after = [ "ugreen-probe-leds.service" ];
            requires = [ "ugreen-probe-leds.service" ];
            serviceConfig = {
              ExecStart = "${package}/bin/ugreen-diskiomon";
              StandardOutput = "journal";
              Restart = "on-failure";
              RestartSec = "5s";
            };
            wantedBy = [ "multi-user.target" ];
            restartTriggers = [
              package
              configFileContent
            ];
          };
        })

        (mkIf cfg.networkMonitor.enable (
          listToAttrs (
            map (interface: {
              name = "ugreen-netdevmon@${interface}";
              value = {
                description = "UGREEN LEDs daemon for monitoring netio (of ${interface}) and blinking corresponding LEDs";
                after = [ "ugreen-probe-leds.service" ];
                requires = [ "ugreen-probe-leds.service" ];
                serviceConfig = {
                  ExecStart = "${package}/bin/ugreen-netdevmon %i";
                  StandardOutput = "journal";
                  Restart = "on-failure";
                  RestartSec = "5s";
                };
                wantedBy = [ "multi-user.target" ];
                restartTriggers = [
                  package
                  configFileContent
                ];
              };
            }) cfg.networkMonitor.interfaces
          )
        ))
      ];

      environment.etc."ugreen-leds.conf" = {
        text = configFileContent;
        mode = "0644";
      };
    }
  );
}
