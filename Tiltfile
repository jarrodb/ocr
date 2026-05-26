load('ext://helm_resource', 'helm_resource')
load('ext://namespace', 'namespace_create')

allow_k8s_contexts('kind-jarrodb')

namespace_create('ocr')

update_settings(k8s_upsert_timeout_secs=120)

docker_build(
    'ghcr.io/jarrodb/ocr',
    '.',
    dockerfile='./Dockerfile',
    live_update=[
        sync('./pkg', '/app/pkg'),
        sync('./cmd', '/app/cmd'),
        run('go build -o /ocr-mock cmd/main.go', trigger=['./go.mod', './go.sum']),
    ],
)

k8s_yaml(helm(
    './charts/ocr',
    name='ocr',
    namespace='ocr',
    set=[
        'image.repository=ghcr.io/jarrodb/ocr',
        'image.tag=latest',
        'gateway.enabled=true',
        'gateway.hostname=ocr.46labs.test',
    ],
))

k8s_resource('ocr', port_forwards=['8080:8080'], labels=['ocr'])
