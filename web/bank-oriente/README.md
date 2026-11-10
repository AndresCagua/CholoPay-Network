# Panel web de Banco Oriente

Lee `GET {VITE_TRANSFER_API_BASE_URL}/v1/participants/bank-oriente/transfers` mediante `src/transfer.ts`. Sin una URL base se usa el adaptador sintético de solo lectura para ejercitar el mismo límite de la API pública; impone el alcance del participante y no accede a bases de datos, libros mayores ni al sistema de archivos. Los estados de carga, error y ausencia de datos son explícitos.

Verificación de Docker: `docker build -t cholopay-bank-oriente-web ./web/bank-oriente`.
