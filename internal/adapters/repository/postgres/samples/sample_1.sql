DO $$
DECLARE
    p_id UUID;
    temp_id UUID;
    voter_ip TEXT;
    user_id UUID;

    -- Storage for generated IDs
    all_created_poll_ids UUID[] := ARRAY[]::UUID[];
    all_created_option_ids UUID[] := ARRAY[]::UUID[]; -- Flat array of all options
    all_created_user_ids UUID[] := ARRAY[]::UUID[];

    -- Poll Data
    titles TEXT[] := ARRAY[
        'Best Programming Language for Backend 2025',
        'Preferred Coffee Style',
        'Ideal Work Environment',
        'Editor of Choice',
        'Primary Operating System',
        'Best Pet Companion',
        'Morning or Night Person?',
        'Preferred Vacation Spot',
        'Superpower of Choice',
        'Tabs vs Spaces'
    ];

    descriptions TEXT[] := ARRAY[
        'Which language do you prefer for scalable server-side development?',
        'How do you take your caffeine in the morning?',
        'Where are you most productive?',
        'What tool do you write code in daily?',
        'What runs on your daily driver?',
        'Who is your furry (or scaly) friend?',
        'When are you most active?',
        'Where do you relax best?',
        'If you could choose one ability.',
        'The endless debate.'
    ];

    -- Options flattened (3 per poll)
    all_options_text TEXT[] := ARRAY[
        'Go', 'Rust', 'Python',
        'Black / Americano', 'Latte / Cappuccino', 'Espresso',
        'Fully Remote', 'Hybrid', 'Office',
        'VS Code', 'IntelliJ / GoLand', 'Vim / Neovim',
        'Linux', 'macOS', 'Windows',
        'Dog', 'Cat', 'Capybara',
        'Early Bird', 'Night Owl', 'Permanently Exhausted',
        'Beach', 'Mountains', 'City Trip',
        'Flight', 'Invisibility', 'Teleportation',
        'Tabs', 'Spaces', 'Mixed (Chaos)'
    ];

    -- User data
    emails TEXT[] := ARRAY[
        'alice@example.com', 'bob@example.com', 'charlie@example.com',
        'diana@example.com', 'eve@example.com', 'frank@example.com',
        'grace@example.com', 'henry@example.com', 'ivy@example.com', 'jack@example.com'
    ];

    names TEXT[] := ARRAY[
        'Alice Johnson', 'Bob Smith', 'Charlie Brown',
        'Diana Prince', 'Eve Williams', 'Frank Miller',
        'Grace Lee', 'Henry Davis', 'Ivy Chen', 'Jack Wilson'
    ];

    i INT;
    j INT;
    opt_idx INT := 1;

    random_poll_idx INT;
    random_opt_offset INT;
    random_user_idx INT;
    target_option_idx INT;
BEGIN
    -- 1. Create Users
    FOR i IN 1..10 LOOP
        INSERT INTO users (email, name)
        VALUES (emails[i], names[i])
        RETURNING id INTO temp_id;

        all_created_user_ids := array_append(all_created_user_ids, temp_id);
    END LOOP;

    -- 2. Create Polls and Options
    FOR i IN 1..10 LOOP
        -- Create Poll
        INSERT INTO polls (title, description)
        VALUES (titles[i], descriptions[i])
        RETURNING id INTO p_id;

        -- Store Poll ID
        all_created_poll_ids := array_append(all_created_poll_ids, p_id);

        -- Create 3 Options
        FOR j IN 1..3 LOOP
            INSERT INTO poll_options (poll_id, text)
            VALUES (p_id, all_options_text[opt_idx])
            RETURNING id INTO temp_id;

            -- Store Option ID in flat array
            all_created_option_ids := array_append(all_created_option_ids, temp_id);
            opt_idx := opt_idx + 1;
        END LOOP;
    END LOOP;

    -- 3. Distribute Votes Randomly across ALL created polls
    -- We'll cast 50 votes randomly distributed
    FOR i IN 1..50 LOOP
        -- Pick a random poll index (1 to 10)
        random_poll_idx := floor(random() * 10) + 1;
        p_id := all_created_poll_ids[random_poll_idx];

        -- Pick a random option for that poll (0 to 2 offset)
        random_opt_offset := floor(random() * 3);

        -- Calculate index in the flat option array
        -- Formula: (poll_index - 1) * 3 + 1 + offset
        -- e.g. Poll 1 (idx 1) -> (0)*3 + 1 + 0 = 1. Correct.
        target_option_idx := (random_poll_idx - 1) * 3 + 1 + random_opt_offset;

        temp_id := all_created_option_ids[target_option_idx];

        voter_ip := '10.0.' || floor(random() * 255) || '.' || floor(random() * 255);

        -- Pick a random user
        random_user_idx := floor(random() * 10) + 1;
        user_id := all_created_user_ids[random_user_idx];

        -- Insert Vote
        -- Ignore conflicts if user already voted on this poll
        BEGIN
            INSERT INTO votes (poll_id, option_id, voter_ip, user_id)
            VALUES (p_id, temp_id, voter_ip::INET, user_id);
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, just skip this vote
        END;
    END LOOP;
END $$;
