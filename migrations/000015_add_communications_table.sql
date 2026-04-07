-- Migration to add communications table for Unified Inbox
CREATE TABLE communications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL, -- whatsapp, email, sms
    direction VARCHAR(10) NOT NULL, -- inbound, outbound
    message TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'received',
    priority VARCHAR(10) DEFAULT 'medium',
    category VARCHAR(50), -- AI classified category
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add sample data for first tenant
INSERT INTO communications (tenant_id, patient_id, channel, direction, message, status, priority, category)
SELECT 
    '00000000-0000-0000-0000-000000000001', 
    id, 
    'whatsapp', 
    'inbound', 
    'Hi, I need to reschedule my appointment for tomorrow.', 
    'received', 
    'high', 
    'booking request'
FROM patients 
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
LIMIT 1;
