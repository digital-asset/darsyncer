{ pkgs, ci }:
let
  requiredPackages = with pkgs; ([
    # these packages are required both in CI and for local development
    crane
    gcrane
    go
    ko
    openjdk21
    (google-cloud-sdk.withExtraComponents [ google-cloud-sdk.components.gke-gcloud-auth-plugin ])
  ]);

in
pkgs.mkShell {
  packages = requiredPackages;
}
