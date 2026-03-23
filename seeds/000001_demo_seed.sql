-- Pre-declare UUIDs for associations
-- Demo Tenant: e830c33a-d04b-4888-91ed-846114eb16eb
-- Admin User:  62ac39e8-b7c1-4bcf-abe5-af80164c9d92
-- Doctor User: b3f20d0f-48cd-47d3-82c5-515afcccd0eb
-- Doctor Obj:  1a3de5ee-c511-4a41-ac4d-8c4d29323c2a

INSERT INTO tenants (id, subdomain, name, primary_color, secondary_color, border_radius, font_family)
VALUES 
('e830c33a-d04b-4888-91ed-846114eb16eb', 'demo', 'Demo Clinic Enterprise', '#059669', '#db2777', '0.5rem', 'Inter');

INSERT INTO users (id, tenant_id, name, email, password_hash, role)
VALUES 
('62ac39e8-b7c1-4bcf-abe5-af80164c9d92', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'Admin Demo', 'admin@demo.com', '$2a$10$w81.m/P55lDqX2pGjE.nruE/1h4D1F/eBAsWk8CgBqJvFvqj1r2Ea', 'admin'), -- hash is 'secret123'
('b3f20d0f-48cd-47d3-82c5-515afcccd0eb', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'Dr. Smith', 'doctor@demo.com', '$2a$10$w81.m/P55lDqX2pGjE.nruE/1h4D1F/eBAsWk8CgBqJvFvqj1r2Ea', 'doctor');

INSERT INTO doctors (id, tenant_id, user_id, full_name, specialty, license_number)
VALUES
('1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'b3f20d0f-48cd-47d3-82c5-515afcccd0eb', 'Dr. Sarah Smith', 'Cardiology', 'MD-100293');

INSERT INTO doctor_availability (tenant_id, doctor_id, day_of_week, start_time, end_time)
VALUES
('e830c33a-d04b-4888-91ed-846114eb16eb', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', 1, '09:00:00', '17:00:00'), -- Monday
('e830c33a-d04b-4888-91ed-846114eb16eb', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', 2, '09:00:00', '17:00:00'), -- Tuesday
('e830c33a-d04b-4888-91ed-846114eb16eb', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', 3, '09:00:00', '17:00:00'), -- Wednesday
('e830c33a-d04b-4888-91ed-846114eb16eb', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', 4, '09:00:00', '17:00:00'), -- Thursday
('e830c33a-d04b-4888-91ed-846114eb16eb', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', 5, '09:00:00', '14:00:00'); -- Friday (Closes early)

-- Pre-declare Patient UUIDs
-- P1: 2b8c9d0d-fed4-4a2b-8a9d-5ac2b4d91e8f
-- P2: 3c9d0e1e-afe5-5b3c-9b0e-6bd3c5ea2f90
-- P3: 4d0e1f2f-b0f6-6c4d-0c1f-7ce4d6fb30a1
-- P4: 5e1f2030-c107-7d5e-1d20-8df5e70c41b2
-- P5: 6f203141-d218-8e6f-2e31-9e06f81d52c3

INSERT INTO patients (id, tenant_id, first_name, last_name, email) VALUES
('2b8c9d0d-fed4-4a2b-8a9d-5ac2b4d91e8f', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'John', 'Doe', 'john@example.com'),
('3c9d0e1e-afe5-5b3c-9b0e-6bd3c5ea2f90', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'Jane', 'Smith', 'jane@example.com'),
('4d0e1f2f-b0f6-6c4d-0c1f-7ce4d6fb30a1', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'Alice', 'Johnson', 'alice@example.com'),
('5e1f2030-c107-7d5e-1d20-8df5e70c41b2', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'Bob', 'Brown', 'bob@example.com'),
('6f203141-d218-8e6f-2e31-9e06f81d52c3', 'e830c33a-d04b-4888-91ed-846114eb16eb', 'Charlie', 'Davis', 'charlie@example.com');

INSERT INTO appointments (tenant_id, patient_id, doctor_id, start_time, end_time, status) VALUES
('e830c33a-d04b-4888-91ed-846114eb16eb', '2b8c9d0d-fed4-4a2b-8a9d-5ac2b4d91e8f', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', NOW() + INTERVAL '1 day', NOW() + INTERVAL '1 day 1 hour', 'scheduled'),
('e830c33a-d04b-4888-91ed-846114eb16eb', '3c9d0e1e-afe5-5b3c-9b0e-6bd3c5ea2f90', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', NOW() + INTERVAL '2 day', NOW() + INTERVAL '2 day 1 hour', 'confirmed'),
('e830c33a-d04b-4888-91ed-846114eb16eb', '4d0e1f2f-b0f6-6c4d-0c1f-7ce4d6fb30a1', '1a3de5ee-c511-4a41-ac4d-8c4d29323c2a', NOW() + INTERVAL '3 day', NOW() + INTERVAL '3 day 1 hour', 'scheduled');
