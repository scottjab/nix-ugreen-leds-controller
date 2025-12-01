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

    environment.systemPackages = [
      package
      pkgs.smartmontools  # For smartctl command used by ugreen-diskiomon
      pkgs.iproute2       # For ping and ip commands used by ugreen-netdevmon
      pkgs.bc             # For bc calculator used by ugreen-netdevmon
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
          restartTriggers = [ package ];
          environment = mkIf (cfg.diskMonitor.configFile != null) {
            UGREEN_LEDS_CONF = toString cfg.diskMonitor.configFile;
          };
        };
      })

      (mkIf cfg.networkMonitor.enable (
        listToAttrs (map (interface: {
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
            restartTriggers = [ package ];
          };
        }) cfg.networkMonitor.interfaces)
      ))
    ];

    environment.etc."ugreen-leds.conf" = {
      source = "${package}/share/ugreen-leds-controller/ugreen-leds.conf";
      mode = "0644";
    };
  };
}
