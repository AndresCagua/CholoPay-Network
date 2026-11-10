# Panel web del operador

El panel lee `GET {VITE_TRANSFER_API_BASE_URL}/v1/participants/{participant_id}/transfers` mediante el cliente limitado al participante en `src/transferFixture.ts`. La respuesta se filtra por el participante solicitado como defensa en profundidad del contrato del adaptador; la autorización de producción permanece del lado del servidor. Si no se configura una URL base, el cliente usa su adaptador sintético documentado de lectura (`synthetic://transfer-read`) para poder ejercitar el límite HTTP público sin acceso del navegador a bases de datos, libros mayores ni al sistema de archivos.

Verificación de Docker: `docker build -t cholopay-operator-web ./web/operator`.
