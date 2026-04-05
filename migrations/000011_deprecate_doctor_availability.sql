-- Migration: 000011_deprecate_doctor_availability.sql
--
-- Renames the legacy `doctor_availability` table to `doctor_availability_legacy`.
-- This table was superseded by the `doctor_schedule` / `doctor_breaks` / `doctor_exceptions`
-- system introduced in migration 000008. Nothing in the application reads from or writes
-- to `doctor_availability` anymore — `doctor_schedule` is the single source of truth.
--
-- We rename rather than drop to:
--   1. Preserve historical data that may exist in the old table.
--   2. Allow a safe rollback path (rename back) if any unexpected consumer is found.
--   3. Defer the hard DROP to a future cleanup migration once stability is confirmed.
--
-- Rollback:
--   ALTER TABLE IF EXISTS doctor_availability_legacy RENAME TO doctor_availability;

ALTER TABLE IF EXISTS doctor_availability RENAME TO doctor_availability_legacy;

COMMENT ON TABLE doctor_availability_legacy IS
  'DEPRECATED — superseded by doctor_schedule (migration 000008). '
  'Kept for historical data preservation. Do not read from or write to this table. '
  'Scheduled for DROP once stability is confirmed.';
