{
  lib,
  stdenv,
  fetchFromGitHub,
  gcc,
  kernel,
  kmod,
  i2c-tools,
  which,
  dmidecode,
  gawk,
  gnused,
  perl,
  util-linux,
  smartmontools,
  zfs,
  iproute2,
  bc,
}:

let
  version = "0.3";
  src = fetchFromGitHub {
    owner = "miskcoo";
    repo = "ugreen_leds_controller";
    rev = "v${version}";
    sha256 = "sha256-eSTOUHs4y6n4cacpjQAp4JIfyu40aBJEMsvuCN6RFZc=";
  };

  # Build the CLI tool
  cli = stdenv.mkDerivation {
    pname = "ugreen-leds-cli";
    inherit version src;

    sourceRoot = "source/cli";

    nativeBuildInputs = [ gcc ];

    buildPhase = ''
      make CC=${gcc}/bin/g++ CFLAGS="-I. -O2 -Wall"
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp ugreen_leds_cli $out/bin/
    '';
  };

  # Build optional C++ utilities
  blink-disk = stdenv.mkDerivation {
    pname = "ugreen-blink-disk";
    inherit version src;

    sourceRoot = "source/scripts";

    nativeBuildInputs = [ gcc ];

    buildPhase = ''
      ${gcc}/bin/g++ -std=c++17 -O2 blink-disk.cpp -o ugreen-blink-disk
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp ugreen-blink-disk $out/bin/
    '';
  };

  check-standby = stdenv.mkDerivation {
    pname = "ugreen-check-standby";
    inherit version src;

    sourceRoot = "source/scripts";

    nativeBuildInputs = [ gcc ];

    buildPhase = ''
      ${gcc}/bin/g++ -std=c++17 -O2 check-standby.cpp -o ugreen-check-standby
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp ugreen-check-standby $out/bin/
    '';
  };

  # Build kernel module
  kernelModule = stdenv.mkDerivation {
    pname = "ugreen-leds-kmod";
    inherit version src;

    sourceRoot = "source/kmod";

    nativeBuildInputs = kernel.moduleBuildDependencies;

    KERNELRELEASE = kernel.modDirVersion;
    KDIR = "${kernel.dev}/lib/modules/${kernel.modDirVersion}/build";

    buildPhase = ''
      make -C $KDIR M=$(pwd) modules
    '';

    installPhase = ''
      # Install in extra directory for boot.extraModulePackages
      mkdir -p $out/lib/modules/${kernel.modDirVersion}/extra
      cp led-ugreen.ko $out/lib/modules/${kernel.modDirVersion}/extra/
    '';

    meta = {
      description = "Kernel module for UGREEN NAS LED controller";
      license = lib.licenses.mit;
      platforms = lib.platforms.linux;
    };
  };

  # Shell scripts wrapper
  scripts = stdenv.mkDerivation {
    pname = "ugreen-leds-scripts";
    inherit version src;

    sourceRoot = "source/scripts";

    nativeBuildInputs = [ kmod i2c-tools which dmidecode gawk gnused perl util-linux smartmontools zfs iproute2 bc ];

    installPhase = ''
      mkdir -p $out/bin
      
      # Use our fixed ugreen-diskiomon script instead of upstream
      cp ${./ugreen-diskiomon-fixed.sh} $out/bin/ugreen-diskiomon
      cp ugreen-netdevmon ugreen-probe-leds $out/bin/

      # Patch scripts to use absolute paths to all required utilities
      # Use perl for all replacements to avoid sed quoting issues
      for script in $out/bin/*; do
        # Kernel module utilities
        ${perl}/bin/perl -i -pe "s|\blsmod\b|${kmod}/bin/lsmod|g" "$script"
        ${perl}/bin/perl -i -pe "s|\bmodprobe\b|${kmod}/bin/modprobe|g" "$script"
        
        # I2C utilities
        ${perl}/bin/perl -i -pe "s|\bi2cdetect\b|${i2c-tools}/bin/i2cdetect|g" "$script"
        
        # System utilities
        ${perl}/bin/perl -i -pe "s|\bwhich\b|${which}/bin/which|g" "$script"
        ${perl}/bin/perl -i -pe "s|\bdmidecode\b|${dmidecode}/bin/dmidecode|g" "$script"
        ${perl}/bin/perl -i -pe "s|\bawk\b|${gawk}/bin/awk|g" "$script"
        ${perl}/bin/perl -i -pe "s|\bsed\b|${gnused}/bin/sed|g" "$script"
        ${perl}/bin/perl -i -pe "s|\blsblk\b|${util-linux}/bin/lsblk|g" "$script"
        # Patch smartctl - handle absolute path first to avoid double paths
        ${perl}/bin/perl -i -pe "s|/usr/sbin/smartctl|${smartmontools}/bin/smartctl|g" "$script"
        # Then patch bare smartctl command
        ${perl}/bin/perl -i -pe "s|\bsmartctl\b|${smartmontools}/bin/smartctl|g" "$script"
        ${perl}/bin/perl -i -pe "s|\bzpool\b|${zfs}/bin/zpool|g" "$script"
        # Patch xargs (used in fixed script)
        ${perl}/bin/perl -i -pe "s|\bxargs\b|${gnused}/bin/xargs|g" "$script"
        # Network utilities (for ugreen-netdevmon)
        ${perl}/bin/perl -i -pe "s|\bping\b|${iproute2}/bin/ping|g" "$script"
        ${perl}/bin/perl -i -pe "s|\bip\b|${iproute2}/bin/ip|g" "$script"
        # Calculator (for ugreen-netdevmon)
        ${perl}/bin/perl -i -pe "s|\bbc\b|${bc}/bin/bc|g" "$script"
      done

      chmod +x $out/bin/*
    '';
  };

in
stdenv.mkDerivation {
  pname = "ugreen-leds-controller";
  inherit version src;

  dontBuild = true;

  installPhase = ''
    mkdir -p $out/bin
    mkdir -p $out/share/ugreen-leds-controller

    # Install CLI
    cp ${cli}/bin/ugreen_leds_cli $out/bin/

    # Install optional utilities
    cp ${blink-disk}/bin/ugreen-blink-disk $out/bin/
    cp ${check-standby}/bin/ugreen-check-standby $out/bin/

    # Install scripts
    cp ${scripts}/bin/* $out/bin/

    # Install config file
    cp ${src}/scripts/ugreen-leds.conf $out/share/ugreen-leds-controller/

    # Install systemd services
    mkdir -p $out/share/systemd/system
    cp ${src}/scripts/systemd/*.service $out/share/systemd/system/
  '';

  passthru = {
    inherit
      cli
      kernelModule
      blink-disk
      check-standby
      scripts
      ;
  };

  meta = {
    description = "LED Controller for UGREEN's DX/DXP NAS Series";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
