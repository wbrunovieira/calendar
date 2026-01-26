-- Add display_order field to events table for drag and drop ordering
ALTER TABLE events ADD COLUMN display_order INTEGER DEFAULT 0;

-- Create index for ordering
CREATE INDEX idx_events_display_order ON events(display_order);

-- Set initial order based on created_at for existing habits
UPDATE events 
SET display_order = subquery.row_num 
FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY calendar_id, event_type ORDER BY created_at) as row_num
    FROM events
    WHERE event_type IN ('HABIT', 'TODO')
) AS subquery
WHERE events.id = subquery.id;
