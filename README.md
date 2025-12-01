# Nix Package and NixOS Module for UGREEN LEDs Controller

Nix package and NixOS module for the [UGREEN LEDs Controller](https://github.com/miskcoo/ugreen_leds_controller), which controls LED lights on UGREEN's DX/DXP NAS Series devices.

## Usage

### Flake

Add to your `flake.nix`:

```nix
{
  inputs.ugreen-leds.url = "github:your-username/nix-ugreen-leds-controller";
  
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

Default config: `/etc/ugreen-leds.conf`. Custom path via `services.ugreen-leds.diskMonitor.configFile`.

See the [original repository](https://github.com/miskcoo/ugreen_leds_controller) for details.

## Requirements

- Linux kernel with I2C support
- `i2c-dev` kernel module
- Root permissions

## License

MIT

