# ptch - Protecode Helper

1. Create *config.yaml* in `$HOME/.ptch` or `$HOME/.config/ptch` with keys:
    - group
    - apiEndpoint
    - token 
1. Use `-r` flag if the particular image is remote one, stored in e.g. GCR. `ptch` will not download it then, relying on Protecode API in its entirety. If it's not set the local image will be stored in temporary location as *.tar* and sent to Protecode.
1. Use `-i` flag to set image to scan, e.g. *gcr.io/kaniko-project/executor:v1.0.0*.
1. Get scan report URL.
1. Profit!