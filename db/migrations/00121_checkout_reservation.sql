-- +goose Up
-- D-031, §14.3/§14.5: failed synchronous attempts keep the provider key stable.
-- Existing encrypted results already have durable provider completion evidence.
ALTER TABLE billing_sessions ADD COLUMN fulfillment_state TEXT NOT NULL DEFAULT 'pending'
 CHECK(fulfillment_state IN ('pending','failed','succeeded'));
UPDATE billing_sessions SET fulfillment_state='succeeded' WHERE encrypted_response IS NOT NULL;
