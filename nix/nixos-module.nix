{ self }:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.notes-web;
  system = pkgs.stdenv.hostPlatform.system;

  command = lib.escapeShellArgs (
    [
      "${cfg.package}/bin/notes-web"
      "-vault"
      cfg.vault
      "-host"
      cfg.host
      "-port"
      (toString cfg.port)
    ]
    ++ lib.optionals (cfg.auth.user != null) [
      "-user"
      cfg.auth.user
    ]
    ++ lib.optionals (cfg.auth.passwordEnv != null) [
      "-password-env"
      cfg.auth.passwordEnv
    ]
    ++ cfg.extraArgs
  );

  commonService = {
    description = "notes-web Markdown/Obsidian web viewer";
    documentation = [ "https://github.com/arkan/notes-web" ];
    after = [ "network.target" ];

    serviceConfig = {
      ExecStart = command;
      Restart = "on-failure";
      RestartSec = "5s";
      NoNewPrivileges = true;
      PrivateTmp = true;
    }
    // lib.optionalAttrs (cfg.environmentFile != null) {
      EnvironmentFile = cfg.environmentFile;
    };
  };

  systemService = commonService // {
    wantedBy = [ "multi-user.target" ];
    serviceConfig = commonService.serviceConfig // {
      User = cfg.user;
      Group = cfg.group;
      ProtectSystem = "strict";
      ProtectHome = "read-only";
      ReadOnlyPaths = [ cfg.vault ];
    };
  };

  userService = commonService // {
    wantedBy = [ "default.target" ];
    unitConfig.ConditionUser = cfg.user;
    serviceConfig = commonService.serviceConfig // {
      ProtectSystem = "strict";
      ReadOnlyPaths = [ cfg.vault ];
    };
  };
in
{
  options.services.notes-web = {
    enable = lib.mkEnableOption "notes-web Markdown/Obsidian web viewer";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${system}.default;
      defaultText = lib.literalExpression "inputs.notes-web.packages.\${pkgs.stdenv.hostPlatform.system}.default";
      description = "Package providing the notes-web binary.";
    };

    mode = lib.mkOption {
      type = lib.types.enum [ "system" "user" ];
      default = "system";
      description = ''
        Whether to run notes-web as a system service or as a systemd user service.

        `system` creates `systemd.services.notes-web` and runs as `services.notes-web.user`.
        `user` creates `systemd.user.services.notes-web` guarded by `ConditionUser`, so it
        runs only in the configured user's user manager.
      '';
    };

    vault = lib.mkOption {
      type = lib.types.str;
      example = "/home/alice/Notes";
      description = "Absolute path to the Markdown/Obsidian vault to serve.";
    };

    host = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1";
      example = "0.0.0.0";
      description = "Address notes-web should bind to.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "TCP port notes-web should listen on.";
    };

    user = lib.mkOption {
      type = lib.types.str;
      default = "notes-web";
      example = "alice";
      description = ''
        User associated with the service.

        For `mode = "system"`, this is the Unix user used by the system service.
        For `mode = "user"`, this is the user selected by the systemd `ConditionUser` guard.
      '';
    };

    group = lib.mkOption {
      type = lib.types.str;
      default = "notes-web";
      example = "users";
      description = "Group used when running as a system service.";
    };

    createUser = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = ''
        Create `services.notes-web.user` and `services.notes-web.group` for system service mode.
        Disable this when running as an existing user that already owns the vault.
      '';
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Open the configured TCP port in the NixOS firewall.";
    };

    extraArgs = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "-some-future-flag" "value" ];
      description = "Additional command-line arguments passed to notes-web.";
    };

    environmentFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      example = "/run/secrets/notes-web.env";
      description = ''
        Optional environment file loaded by systemd. Use this for secrets such as the
        variable named by `services.notes-web.auth.passwordEnv`.
      '';
    };

    auth = {
      user = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        example = "florian";
        description = "Optional HTTP Basic Auth username passed with `-user`.";
      };

      passwordEnv = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        example = "NOTES_WEB_PASSWORD";
        description = "Optional environment variable name containing the HTTP Basic Auth password.";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = lib.hasPrefix "/" cfg.vault;
        message = "services.notes-web.vault must be an absolute path.";
      }
      {
        assertion = cfg.mode != "user" || !cfg.createUser;
        message = "services.notes-web.createUser must be false when mode = \"user\".";
      }
      {
        assertion = (cfg.auth.user == null) == (cfg.auth.passwordEnv == null);
        message = "services.notes-web.auth.user and auth.passwordEnv must be configured together.";
      }
    ];

    users.users = lib.mkIf (cfg.mode == "system" && cfg.createUser) {
      ${cfg.user} = {
        isSystemUser = true;
        group = cfg.group;
      };
    };

    users.groups = lib.mkIf (cfg.mode == "system" && cfg.createUser) {
      ${cfg.group} = { };
    };

    systemd.services.notes-web = lib.mkIf (cfg.mode == "system") systemService;
    systemd.user.services.notes-web = lib.mkIf (cfg.mode == "user") userService;

    networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [ cfg.port ];
  };
}
