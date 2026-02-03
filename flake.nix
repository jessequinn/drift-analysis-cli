{
  description = "ZPE Systems DevSecOps DevShell";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }:
    let
      # Supported systems
      systems = [ "x86_64-darwin" "aarch64-darwin" "x86_64-linux" ];

      # Shared shellHook
      shellHook = let envDir = toString ./.; in ''
        echo "========================================"

        # Load .env if present
        ENV_FILE="${envDir}/.env"
        if [ -f "$ENV_FILE" ]; then
          set -a
          source "$ENV_FILE"
          set +a
          echo "Loaded environment variables from $ENV_FILE"
        fi

        echo "Environment loaded successfully!"
        echo "========================================"
      '';
    in
    {
      devShells = builtins.listToAttrs (map (system:
        let
          # Import pkgs for this system
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };

          # Base packages
          basePackages = with pkgs; [
            nodejs_24
            python3
            python3Packages.pip
            go
            errcheck
            ineffassign
            gosec
            gotools
            go-tools # Includes staticcheck
            gofumpt
            golangci-lint
            git
            gh
            github-copilot-cli
            claude-code
            gnupg
            openssl
            openssh
            nano
            vim-full
            vimPlugins.vim-gnupg
            coreutils
            diffutils
            findutils
            gnugrep
            gnused
            gawk
            curl
            wget
            jq
            yq
            tree
            htop
            shellcheck
            pylint
            ruff
            eslint_d
            tflint
            hadolint
            black
            isort
            nixpkgs-fmt
            shfmt
            nodePackages.prettier
            gocyclo
            # Network troubleshooting tools
            tcpdump
            netcat-gnu
            nmap
            bind # provides dig, nslookup, host
            mtr
            iperf3
            socat
          ];

          # Security packages
          securityPackages = with pkgs; [
            syft
            grype
            trivy
            gitleaks
            trufflehog
            semgrep
            govulncheck
            # Added for compose-provisioner project
            protobuf # Protocol Buffers compiler (protoc)
            protoc-gen-go # Go protobuf plugin
            protoc-gen-go-grpc # Go gRPC plugin
          ];

          # IaC / K8s / Cloud
          iacPackages = with pkgs; [
            terraform
            opentofu
            ansible
            kubectl
            kubernetes-helm
            k9s
            minikube
            kubectx
            kustomize
            kor
            popeye
            google-cloud-sdk
          ];

          # Container tools (macOS-safe)
          containerPackages = with pkgs; [
            dive
          ];

          # Minimal packages
          minimalPackages = with pkgs; [
            git
            nodejs_24
            python3
            go
            coreutils
          ];
        in
        {
          name = system;
          value = {
            # Full shell
            default = pkgs.mkShell {
              packages = basePackages ++ securityPackages ++ iacPackages ++ containerPackages;
              shellHook = shellHook;
            };

            # Security-only shell
            security = pkgs.mkShell {
              packages = basePackages ++ securityPackages;
              shellHook = shellHook;
            };

            # K8s / IaC shell
            k8s = pkgs.mkShell {
              packages = basePackages ++ iacPackages;
              shellHook = shellHook;
            };

            # Minimal shell
            minimal = pkgs.mkShell {
              packages = minimalPackages;
              shellHook = shellHook;
            };
          };
        }) systems);
    };
}
