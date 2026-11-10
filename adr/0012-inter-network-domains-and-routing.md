# ADR 0012: dominios y enrutamiento interredes de CholoPay

- Estado: Aceptado para el primer slice de dos redes
- Fecha: 2026-11-09
- Base de referencia: BCE Codificación JPRM-2025-006-M y BCE-GG-008-2025, consultadas el 2026-11-09

## Contexto

El producto ampliado simula dos redes de pago independientes: una red bancaria que contiene Costa, Sierra y Oriente, y una red fintech que contiene Wallet A y Wallet B. CholoNet debe enrutar instrucciones sin colapsar sus límites de propiedad.

## Decisión

Persistir centralmente los dominios de red y la pertenencia de participantes. Cada participante pertenece exactamente a un dominio activo en un instante dado. La decisión de ruta compara los dominios de origen y destino:

- mismo dominio: `INTRA_NETWORK`, gestionado por la política de esa red;
- dominios diferentes: `INTER_NETWORK`, enviado mediante la coordinación interredes de CholoNet;
- pertenencia inexistente, suspendida o incompatible: rechazar antes de reservar.

La decisión de ruta es informativa y no establece la liquidación. El ledger central de liquidación sigue siendo la única fuente de `SETTLED`. La resolución del directorio, la autorización, las reglas de riesgo, la compensación y los ledgers de participantes siguen siendo preocupaciones separadas.

## Consecuencias

El modelo demuestra el enrutamiento interredes sin afirmar que implemente el SIP/RPI del BCE. Permite credenciales y políticas específicas por dominio, preserva el aislamiento de participantes y deja espacio para adaptadores de directorios distribuidos. No se afirma ISO 20022; los eventos usan la envoltura versionada de CholoPay hasta que se apruebe un perfil concreto.
