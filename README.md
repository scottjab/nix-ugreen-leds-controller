# Nix Package and NixOS Module for UGREEN LEDs Controller

Nix package and NixOS module for the [UGREEN LEDs Controller](https://github.com/miskcoo/ugreen_leds_controller), which controls LED lights on UGREEN's DX/DXP NAS Series devices.

This package includes a unified Go service that replaces the original shell scripts for improved performance and maintainability.

## Usage

### Flake

Add to your `flake.nix`:

```nix
{
  inputs.ugreen-leds.url = "github:scottjab/nix-ugreen-leds-controller";
  
  outputs = { nixpkgs, ugreen-leds, ... }: {
    nixosConfigurations.your-host = nixpkgs.lib.nixosSystem {
      modules = [ ugreen-leds.nixosModules.default ];
    };
  };
}
```

### NixOS Configuration

```nix
{
  services.ugreen-leds = {
    enable = true;
    kernelModule.enable = true;
    probeLeds.enable = true;
    diskMonitor.enable = true;
    networkMonitor = {
      enable = true;
      interfaces = [ "enp2s0" ];
    };
  };
}
```

### Build Package

```bash
nix build .#ugreen-leds-controller
```

## Configuration

Configuration is managed through the NixOS module options and written to `/etc/ugreen-leds.conf`. The Go service reads this configuration file at startup.

See the [original repository](https://github.com/miskcoo/ugreen_leds_controller) for details on the underlying kernel module and hardware support.

## Requirements

- Linux kernel with I2C support
- `i2c-dev` kernel module
- Root permissions

## License

MIT

