-- Enable performance schema (already on via command flag)
-- Create table without index on user_id (same pattern as Postgres test)

CREATE TABLE posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert 100k rows
DELIMITER //
CREATE PROCEDURE populate_posts()
BEGIN
    DECLARE i INT DEFAULT 0;
    WHILE i < 100000 DO
        INSERT INTO posts (user_id, title, content)
        VALUES (
            FLOOR(1 + RAND() * 1000),
            CONCAT('Post Title ', i),
            CONCAT('This is the content for post number ', i)
        );
        SET i = i + 1;
    END WHILE;
END //
DELIMITER ;

CALL populate_posts();
DROP PROCEDURE populate_posts;