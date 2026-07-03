-- Fee categories (per-school configurable)
CREATE TABLE IF NOT EXISTS fee_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (school_id, code)
);

CREATE INDEX IF NOT EXISTS idx_fee_categories_school ON fee_categories(school_id);

-- Seed default categories for existing schools
INSERT INTO fee_categories (id, school_id, name, code)
SELECT uuid_generate_v4(), s.id, v.name, v.code
FROM schools s
CROSS JOIN (
    VALUES
        ('Academic', 'academic'),
        ('Uniform', 'uniform'),
        ('Materials', 'materials'),
        ('Extra Curricular', 'extra_curricular'),
        ('Other', 'other')
) AS v(name, code)
ON CONFLICT (school_id, code) DO NOTHING;

-- Allow custom category codes on fee types
ALTER TABLE fee_types DROP CONSTRAINT IF EXISTS fee_types_category_check;

-- Invoice send history
CREATE TABLE IF NOT EXISTS invoice_send_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    sent_to VARCHAR(255) NOT NULL,
    send_type VARCHAR(20) NOT NULL CHECK (send_type IN ('initial', 'reminder')),
    sent_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_invoice_send_logs_invoice ON invoice_send_logs(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_send_logs_school ON invoice_send_logs(school_id);

-- Optional bank details for invoice payment slip
ALTER TABLE schools ADD COLUMN IF NOT EXISTS bank_name VARCHAR(100);
ALTER TABLE schools ADD COLUMN IF NOT EXISTS bank_account_name VARCHAR(150);
ALTER TABLE schools ADD COLUMN IF NOT EXISTS bank_account_number VARCHAR(30);
