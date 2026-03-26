-- Add timezone to tenants
ALTER TABLE tenants ADD COLUMN timezone VARCHAR(50) DEFAULT 'Asia/Amman';

-- Update existing tenants to a sensible default if they were empty
UPDATE tenants SET timezone = 'Asia/Amman' WHERE timezone IS NULL;
