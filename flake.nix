{
  inputs.nixpkgs.url = "nixpkgs/nixos-25.11";

  outputs = { self, nixpkgs, ... }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      nixpkgsFor = forAllSystems (system: import nixpkgs {
        inherit system;
        overlays = [ self.overlay ];
      });

    in
    {
      overlay = final: prev: {
        wuziqi = final.buildGoModule {
          pname = "wuziqi";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-s7ZKKPnwRvB/uVnqICZq7k4oLri6Jo+gdKjPmuorHvU=";
        };
      };

      packages = forAllSystems (system:
        {
          inherit (nixpkgsFor.${system}) wuziqi;
        });
      defaultPackage = forAllSystems (system: self.packages.${system}.wuziqi);

      nixosModules.wuziqi = { config, lib, pkgs, ... }:
        with lib;
        let
          cfg = config.services.wuziqi;
        in
        {
                  
        options.services.wuziqi = {
          enable = mkEnableOption "The wuziqi server";

          extraArgs = mkOption {
            type = types.listOf types.str;
            default = [];
            description = "Additional command line arguments";
          };
        };
      
        config = mkIf cfg.enable {
          nixpkgs.overlays = [ self.overlay ];

          users.users.wuziqi = {
            isSystemUser = true;
            group = "wuziqi";
            shell = "/bin/false";
          };
        
          users.groups.wuziqi = {};
        
          systemd.services.wuziqi = {
            enable = true;
            description = "Wuziqi server.";
            wantedBy = [ "multi-user.target" ];
            serviceConfig = {
              User = "wuziqi";
              Group = "wuziqi";
              NoNewPrivileges = true;
              ProtectSystem = "strict";
              ExecStart = "${pkgs.wuziqi}/bin/wuziqi ${escapeShellArgs cfg.extraArgs}";
              ProtectHome = true;
              Restart = "always";
              RestartSec = 5;
            };
          };
        };
      };
    };
}
