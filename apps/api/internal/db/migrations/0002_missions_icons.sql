-- Skill-focused practice + parent-assigned missions ("Latih 5 challenge tentang
-- pembagian minggu ini", §13) and the move from emoji to licensed icon names.

ALTER TABLE assessments ADD COLUMN skill_id int REFERENCES skills (id);

CREATE TABLE child_missions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id      uuid NOT NULL REFERENCES children (id) ON DELETE CASCADE,
    skill_id      int  NOT NULL REFERENCES skills (id),
    target        int  NOT NULL CHECK (target BETWEEN 1 AND 20),
    completed     int  NOT NULL DEFAULT 0,
    status        text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled')),
    created_by    uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    due_at        timestamptz,
    completed_at  timestamptz
);
-- One active mission per skill per child.
CREATE UNIQUE INDEX child_missions_active_idx ON child_missions (child_id, skill_id) WHERE status = 'active';

ALTER TABLE assessments ADD COLUMN mission_id uuid REFERENCES child_missions (id) ON DELETE SET NULL;

-- Emoji avatars → icon avatars.
ALTER TABLE children ALTER COLUMN avatar SET DEFAULT 'cat';
UPDATE children SET avatar = CASE avatar
    WHEN 'fox' THEN 'cat' WHEN 'panda' THEN 'dog' WHEN 'frog' THEN 'rabbit'
    WHEN 'owl' THEN 'bird' WHEN 'turtle' THEN 'butterfly' WHEN 'whale' THEN 'robot' ELSE avatar END;
DELETE FROM child_inventory WHERE item_kind = 'reward' AND item_code IN ('fox','panda','frog','owl','turtle','whale','island','night-sky');
DELETE FROM rewards WHERE code IN ('fox','panda','frog','owl','turtle','whale','island','night-sky');
