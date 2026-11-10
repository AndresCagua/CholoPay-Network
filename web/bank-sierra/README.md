# Panel web de Banco Sierra

Lee `GET {TRANSFER_API_BASE_URL}/v1/participants/bank-sierra/transfers` mediante el cliente limitado al participante en `src/app/transfer-read-model.ts`. El adaptador sintético predeterminado es de solo lectura e impone el participante en su contrato; configurá la base HTTP en tiempo de ejecución mediante `globalThis.__env.TRANSFER_API_BASE_URL`, inyectado. El navegador no accede a bases de datos, libros mayores ni al sistema de archivos. Los estados de carga, error y ausencia de datos se renderizan explícitamente.

Verificación de Docker: `docker build -t cholopay-bank-sierra-web ./web/bank-sierra`.
