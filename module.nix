{
  config,
  lib,
  pkgs,
  ...
}:

with lib;

let
  cfg = config.services.ugreen-leds;
  package = cfg.package;
in
{
  options.services.ugreen-leds = {
    enable = mkEnableOption "UGREEN LEDs controller";

    package = mkOption {
      type = types.package;
      default = pkgs.ugreen-leds-controller;
      defaultText = "pkgs.ugreen-leds-controller";
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
      configFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "Path to custom configuration file. If null, uses default.";
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
    };
  };

  config = mkIf cfg.enable {
    boot.kernelModules = mkIf cfg.kernelModule.enable [
      "i2c-dev"
      "led-ugreen"
      "ledtrig-oneshot"
      "ledtrig-netdev"
    ];

    boot.extraModulePackages = mkIf cfg.kernelModule.enable [
      package.kernelModule
    ];

    environment.systemPackages = [ package ];

    systemd.services.ugreen-probe-leds = mkIf cfg.probeLeds.enable {
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

    # Note: ugreen-power-led script doesn't exist in v0.3
    # systemd.services.ugreen-power-led = mkIf cfg.powerLed.enable {
    #   description = "UGREEN LEDs daemon for configuring power LED";
    #   after = [ "ugreen-probe-leds.service" ];
    #   requires = [ "ugreen-probe-leds.service" ];
    #   serviceConfig = {
    #     Type = "oneshot";
    #     ExecStart = "${package}/bin/ugreen-power-led";
    #     RemainAfterExit = true;
    #     StandardOutput = "journal";
    #   };
    #   wantedBy = [ "multi-user.target" ];
    # };

    systemd.services.ugreen-diskiomon = mkIf cfg.diskMonitor.enable {
      description = "UGREEN LEDs daemon for monitoring diskio and blinking corresponding LEDs";
      after = [ "ugreen-probe-leds.service" ];
      requires = [ "ugreen-probe-leds.service" ];
      serviceConfig = {
        ExecStart = "${package}/bin/ugreen-diskiomon";
        StandardOutput = "journal";
      };
      wantedBy = [ "multi-user.target" ];
      environment = mkIf (cfg.diskMonitor.configFile != null) {
        UGREEN_LEDS_CONF = toString cfg.diskMonitor.configFile;
      };
    };

    systemd.services.ugreen-netdevmon = mkMerge (
      map (interface: {
        "ugreen-netdevmon@${interface}" = {
          description = "UGREEN LEDs daemon for monitoring netio (of ${interface}) and blinking corresponding LEDs";
          after = [ "ugreen-probe-leds.service" ];
          requires = [ "ugreen-probe-leds.service" ];
          serviceConfig = {
            ExecStart = "${package}/bin/ugreen-netdevmon %i";
            StandardOutput = "journal";
          };
          wantedBy = [ "multi-user.target" ];
        };
      }) cfg.networkMonitor.interfaces
    );

    environment.etc."ugreen-leds.conf" = {
      source = "${package}/share/ugreen-leds-controller/ugreen-leds.conf";
      mode = "0644";
    };
  };
}
