# Runbook local de la fase 0

1. Instala Docker Engine y el complemento Compose; no se requieren runtimes de lenguajes, herramientas de compilación, clientes de bases de datos ni servicios del host.
2. Genera la configuración con `docker compose --env-file infra/config.env.template -f infra/compose.yaml config`.
3. Inicia la pila privada con `docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db sierra-db oriente-db bank-core`.
4. Ejecuta las pruebas con `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test`.
5. Ejecuta `./infra/scripts/smoke.sh` to verify health and database isolation from Docker.
6. Detén la pila con `docker compose --env-file infra/config.env.template -f infra/compose.yaml down`; volumes are retained by default.

El entorno Compose predeterminado no vincula puertos del host, por lo que no puede colisionar con servicios del host. Si un perfil futuro publica un puerto, agrega un check de despliegue exclusivo de Docker y verifica su disponibilidad antes de habilitarlo. El entorno Compose es únicamente para datos sintéticos; no agregues credenciales a Git.
