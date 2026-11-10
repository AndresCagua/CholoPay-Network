# Phase 8 — Hardening local y recuperación reproducible
Fuente: `spec.md`, secciones 14, 15, 16, 18, 22 y 27.
Scope: controles locales Docker-only, backups/restauración de datos sintéticos y runbooks; no Kubernetes, GitOps, publicación externa ni modificación de infraestructura externa.

## Specs

1. **S1 — Hardening y operación**
   > políticas, backups, demos públicas protegidas, documentación en inglés, vídeos de demostración.

2. **S2 — Recuperabilidad**
   > entorno recuperable y demostración sin datos reales.

3. **S3 — Calidad y seguridad**
   > logs estructurados; métricas RED/USE; alertas de discrepancias de ledger, pagos atascados, DLQ y errores de firma.
   > Runbooks para caída de participante, reconciliación, replay de outbox, restauración y rotación de credenciales.

4. **S4 — Límites**
   > No introducir Kubernetes hasta que el entorno Compose y las pruebas estén estabilizados.
   > No modificar infraestructura externa sin autorización explícita.
   > No procesar dinero real ni almacenar secretos reales.

## Tasks

- T1 (S1, S4): Añadir comprobaciones locales de configuración segura para Compose, ausencia de puertos publicados no autorizados y secretos reales. Route: inline after exploration. Commit: pending.
- T2 (S2): Crear backup/restauración reproducible de una base sintética mediante Docker, sin tocar infraestructura externa. Route: inline. Commit: pending.
- T3 (S1-S3): Escribir runbooks operativos mínimos para backup, restore, caída de participante y rotación de credenciales sintéticas. Route: inline. Commit: pending.
- T4 (S1-S4): Ejecutar checks Docker-only, validar evidencia y documentar límites. Route: inline. Commit: pending.

## Log

L1. Usuario: "que sigue, continua"
L2. El slice previo implementó CholoLab como laboratorio Docker-only. Se continúa con hardening local y recuperación reproducible; Kubernetes, GitOps, publicación externa y cambios de infraestructura permanecen fuera de alcance.
L3. Implementados `infra/scripts/hardening-check.sh`, `infra/scripts/backup-restore.sh` y `docs/runbooks/phase-8-hardening-recovery.md`. Los scripts validan ausencia de puertos por defecto, bases aisladas, plantillas sin secretos probables y backup/restauración PostgreSQL sintético.
L4. Evidencia: hardening check PASS; `docker compose config` PASS; backup Docker de las cuatro bases aisladas PASS después de esperar healthchecks con `docker compose up --wait`; cleanup de volúmenes completado. El primer intento sin readiness falló por carrera de arranque y se corrigió ejecutando con `--wait`.
