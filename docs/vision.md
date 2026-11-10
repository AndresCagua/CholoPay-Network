# Visión y límites

CholoPay demuestra flujos interoperables de pagos instantáneos con tres bancos simulados independientes, una red central, un directorio de CholoKeys y un ledger de liquidación. Todo el dinero, las identidades, las cuentas, las obligaciones y los saldos son sintéticos.

## Fuera de alcance

No forman parte del alcance la banca real, la interoperabilidad con Pix/SPI/DICT, la certificación regulatoria, los datos personales reales, la infraestructura de tarjetas PAN/CVV ni las dependencias pagas o cerradas.

## Límite de la fase 0

La fase 0 establece la arquitectura escrita, los contratos, el modelo de amenazas, la infraestructura FOSS local, los checks de salud y una verificación smoke reproducible. No implementa interfaces orientadas a clientes ni comportamiento financiero.

## Límites de confianza

Los bancos son propietarios de las cuentas de clientes y de sus ledgers. La red es propietaria del registro de participantes, el directorio de CholoKeys, las cuentas de liquidación y el estado de liquidación. Los facturadores son propietarios de las obligaciones. El gateway nunca escribe en un ledger. Las proyecciones del operador son de solo lectura, salvo que un ADR introduzca una operación administrativa explícitamente auditada.
