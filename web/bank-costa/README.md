# Panel web de Banco Costa

Lee `GET {NEXT_PUBLIC_TRANSFER_API_BASE_URL}/v1/participants/bank-costa/transfers` mediante `app/transfers.ts`. El adaptador sintético es el predeterminado cuando falta la URL base; es de solo lectura y está limitado al participante, y existe únicamente para ejercitar el límite de la API pública hasta que bank-core exponga esta proyección. El navegador nunca accede a una base de datos, un libro mayor ni al sistema de archivos. Las respuestas de carga, error y ausencia de datos son explícitas.

Verificación de Docker: `docker build -t cholopay-bank-costa-web ./web/bank-costa`.
