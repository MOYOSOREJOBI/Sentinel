-- 007: escalation policies and on-call rotation tables
CREATE TABLE IF NOT EXISTS escalation_policies (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    severity_band TEXT NOT NULL CHECK (severity_band IN ('elevated','high','critical')),
    notify_channel TEXT NOT NULL DEFAULT '',
    sla_minutes INT NOT NULL DEFAULT 60,
    on_call_target TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_escalation_policies_severity ON escalation_policies(severity_band);

CREATE TABLE IF NOT EXISTS on_call_rotations (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    current_on_call TEXT NOT NULL,
    rotation_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotation_end TIMESTAMPTZ,
    policy_id UUID REFERENCES escalation_policies(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_on_call_rotations_name ON on_call_rotations(name);
