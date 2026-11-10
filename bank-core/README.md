# Núcleo bancario de CholoPay — Fase 1

Núcleo bancario Spring Boot parametrizado para Banco Costa, Banco Sierra y Banco Oriente. Este primer corte implementa cuentas sintéticas, saldos en centavos enteros, diarios inmutables equilibrados, reservas y transferencias internas idempotentes. No implementa liquidación interbancaria.

## Compilación y pruebas exclusivas para Docker

El host solo necesita Docker Engine y el complemento Compose. Java, Maven, Gradle y PostgreSQL los proporcionan los contenedores del proyecto y no deben instalarse ni invocarse en el host.

From the repository root:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test
```

La imagen de pruebas se construye desde `bank-core/Dockerfile` usando Maven y JDK 21 dentro del contenedor. La imagen de ejecución solo contiene el JRE y la aplicación empaquetada.

Para ejecutar el servicio con la base de datos de Costa completamente dentro de Compose:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db bank-core
```

El servicio solo es accesible para otros servicios Compose en `http://bank-core:8080`; de forma predeterminada no se publica ningún puerto al host. Si más adelante se añade un proxy de desarrollo local, debe comprobarse su puerto del host antes de enlazarlo.

## Comportamiento financiero

- Las cantidades son centavos de USD enteros positivos.
- Un diario debe contener asientos de débito/crédito equilibrados.
- Los diarios publicados son inmutables en el modelo de dominio.
- Las transferencias internas reservan, capturan, adeudan y abonan dentro de una única sección crítica del servicio.
- Reutilizar una clave de idempotencia con la misma cantidad devuelve la transacción original; una cantidad diferente se rechaza.
- Una transferencia interna exitosa no es una liquidación interbancaria.

Las migraciones de PostgreSQL están en `src/main/resources/db/migration`. El adaptador receptor interno se expone en `POST /internal/v1/receiver-credits`; aplica un abono sintético de compensación de red a una cuenta de cliente en una transacción local y reproduce la misma entrega sin un segundo diario. No demuestra la liquidación central ni sustituye el protocolo de confirmación de la red.

El siguiente corte de implementación añade transporte autenticado del adaptador banco-red y mapeo público de errores antes de afirmar un comportamiento similar al de producción.
