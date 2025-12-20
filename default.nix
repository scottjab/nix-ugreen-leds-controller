{
  lib,
  stdenv,
  fetchFromGitHub,
  buildGoModule,
  gcc,
  kernel,
  kmod,
  i2c-tools,
  which,
  dmidecode,
  util-linux,
  smartmontools,
  zfs,
  iproute2,
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

  # Build Go service
  goService = buildGoModule {
    pname = "ugreen-leds-service";
    version = "0.4.0";
    src = lib.cleanSource ./.;

    # No dependencies to vendor
    vendorHash = null;

    subPackages = [ "cmd/ugreen-leds-service" ];

    # Runtime dependencies
    propagatedBuildInputs = [
      smartmontools
      zfs
      iproute2
      util-linux
      dmidecode
    ];

    meta = {
      description = "Go service for UGREEN LEDs monitoring";
      license = lib.licenses.mit;
      platforms = lib.platforms.linux;
    };
  };

  # Build probe-leds script (still needed for hardware initialization)
  probe-leds = stdenv.mkDerivation {
    pname = "ugreen-probe-leds";
    inherit version src;

    sourceRoot = "source/scripts";

    nativeBuildInputs = [
      kmod
      i2c-tools
      which
    ];

    installPhase = ''
      mkdir -p $out/bin
      cp ugreen-probe-leds $out/bin/
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

    # Install Go service
    cp ${goService}/bin/ugreen-leds-service $out/bin/

    # Install probe-leds script
    cp ${probe-leds}/bin/ugreen-probe-leds $out/bin/

    # Install optional utilities
    cp ${blink-disk}/bin/ugreen-blink-disk $out/bin/
    cp ${check-standby}/bin/ugreen-check-standby $out/bin/

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
      goService
      probe-leds
      blink-disk
      check-standby
      ;
  };

  meta = {
    description = "LED Controller for UGREEN's DX/DXP NAS Series";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
