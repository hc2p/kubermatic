# Copyright 2023 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/usr/bin/env bash
set -xeuo pipefail

dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

rm -rf $dir/{templates,crds}
mkdir -p $dir/{templates,crds}

latest_release=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases | grep tag_name | grep -v -- '-rc' | sort -r | head -1 | awk -F': ' '{print $2}' | sed 's/,//' | xargs)
latest_cdi_release=$(basename $(curl -s -w %{redirect_url} https://github.com/kubevirt/containerized-data-importer/releases/latest))

function boilerplate() {
  cat ${dir}/../../hack/boilerplate/ce/boilerplate.yaml.txt | sed -e "s/\<YEAR\>/$(date +'%Y')/"
}

cat << EOF > values.yaml
$(boilerplate)
EOF

cat << EOF > Chart.yaml
$(boilerplate)

apiVersion: v1
name: kubevirt
version: ${latest_release}
appVersion: ${latest_release}
description: KubeVirt chart for KKP local installation 
keywords:
  - kubermatic
  - kubevirt
home: https://github.com/kubevirt/kubevirt
sources:
  - https://github.com/kubermatic/kubermatic
maintainers:
  - name: The Kubermatic Kubernetes Platform contributors
    email: support@kubermatic.com
EOF

wget https://github.com/kubevirt/kubevirt/releases/download/$latest_release/kubevirt-operator.yaml -O ${dir}/templates/kubevirt-operator.yaml
wget https://github.com/kubevirt/kubevirt/releases/download/$latest_release/kubevirt-cr.yaml -O ${dir}/templates/kubevirt-cr.yaml
wget https://github.com/kubevirt/containerized-data-importer/releases/download/$latest_cdi_release/cdi-operator.yaml -O ${dir}/templates/cdi-operator.yaml
wget https://github.com/kubevirt/containerized-data-importer/releases/download/$latest_cdi_release/cdi-cr.yaml -O ${dir}/templates/cdi-cr.yaml

pushd $dir/templates
yq -s '.kind + "-" + .metadata.name + ".yaml"' ./kubevirt-operator.yaml
rm ./kubevirt-operator.yaml

yq -s '.kind + "-" + .metadata.name + ".yaml"' ./cdi-operator.yaml
rm ./cdi-operator.yaml

for f in *; do 
    l=$(echo "$f" | tr '[A-Z:]' '[a-z-]')
    cat <(boilerplate) "$f" > tmp.yaml
    mv tmp.yaml "$f"
    if [[ "$f" != "$l" ]]; then
      mv "$f" "$l"
    fi
done
mv customresourcedefinition-* ../crds/
popd
