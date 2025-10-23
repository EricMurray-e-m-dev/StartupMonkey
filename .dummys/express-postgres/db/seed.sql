-- Seed script to generate test data with performance issues
-- 10k users, 100k posts, 500k comments

-- Generate 10,000 users
INSERT INTO users (username, email, full_name, bio, avatar_url, created_at)
SELECT
    'user_' || generate_series AS username,
    'user_' || generate_series || '@example.com' AS email,
    'User ' || generate_series AS full_name,
    'Bio for user ' || generate_series || '. ' ||
    CASE
        WHEN generate_series % 3 = 0 THEN 'Love writing about technology and startups.'
        WHEN generate_series % 3 = 1 THEN 'Passionate about product development and design.'
        ELSE 'Interested in data science and machine learning.'
    END AS bio,
    'https://avatars.example.com/user_' || generate_series || '.jpg' AS avatar_url,
    CURRENT_TIMESTAMP - (random() * interval '365 days') AS created_at
FROM generate_series(1, 10000);

-- Generate 100,000 posts with realistic distribution
INSERT INTO posts (title, content, summary, author_id, category, tags, view_count, like_count, published, created_at)
SELECT
    CASE (generate_series % 20)
        WHEN 0 THEN 'Getting Started with ' || (ARRAY['React', 'Node.js', 'Python', 'Go', 'Rust'])[1 + (generate_series % 5)]
        WHEN 1 THEN 'Deep Dive into ' || (ARRAY['Microservices', 'Database Design', 'API Architecture', 'DevOps', 'Security'])[1 + (generate_series % 5)]
        WHEN 2 THEN 'Building ' || (ARRAY['Scalable Applications', 'Real-time Systems', 'Mobile Apps', 'Web Platforms', 'AI Tools'])[1 + (generate_series % 5)]
        WHEN 3 THEN 'Understanding ' || (ARRAY['Distributed Systems', 'Cloud Computing', 'Machine Learning', 'Blockchain', 'IoT'])[1 + (generate_series % 5)]
        WHEN 4 THEN 'Best Practices for ' || (ARRAY['Code Review', 'Testing', 'Deployment', 'Monitoring', 'Documentation'])[1 + (generate_series % 5)]
        WHEN 5 THEN 'Introduction to ' || (ARRAY['GraphQL', 'Docker', 'Kubernetes', 'Terraform', 'CI/CD'])[1 + (generate_series % 5)]
        WHEN 6 THEN 'Advanced ' || (ARRAY['JavaScript', 'SQL', 'Algorithms', 'Data Structures', 'System Design'])[1 + (generate_series % 5)]
        WHEN 7 THEN 'Performance Optimization in ' || (ARRAY['Web Applications', 'Databases', 'Mobile Apps', 'APIs', 'Frontend'])[1 + (generate_series % 5)]
        WHEN 8 THEN 'Debugging ' || (ARRAY['Production Issues', 'Memory Leaks', 'Performance Problems', 'Network Issues', 'Race Conditions'])[1 + (generate_series % 5)]
        WHEN 9 THEN 'Scaling ' || (ARRAY['Startup Infrastructure', 'Database Performance', 'API Throughput', 'User Growth', 'Team Processes'])[1 + (generate_series % 5)]
        WHEN 10 THEN 'Common Mistakes in ' || (ARRAY['Software Architecture', 'Database Design', 'API Development', 'Security', 'DevOps'])[1 + (generate_series % 5)]
        WHEN 11 THEN 'How to Choose ' || (ARRAY['The Right Database', 'A Tech Stack', 'Cloud Provider', 'Monitoring Tools', 'Testing Framework'])[1 + (generate_series % 5)]
        WHEN 12 THEN 'Migration Guide: ' || (ARRAY['MySQL to PostgreSQL', 'Monolith to Microservices', 'AWS to GCP', 'REST to GraphQL', 'JavaScript to TypeScript'])[1 + (generate_series % 5)]
        WHEN 13 THEN 'Why ' || (ARRAY['TypeScript', 'PostgreSQL', 'Docker', 'Monitoring', 'Testing'])[1 + (generate_series % 5)] || ' is Essential for Startups'
        WHEN 14 THEN 'Building Your First ' || (ARRAY['REST API', 'React App', 'Mobile App', 'Microservice', 'AI Model'])[1 + (generate_series % 5)]
        WHEN 15 THEN 'Lessons Learned: ' || (ARRAY['Scaling to 1M Users', 'Database Optimization', 'Team Management', 'Product Launch', 'System Outages'])[1 + (generate_series % 5)]
        WHEN 16 THEN 'The Complete Guide to ' || (ARRAY['API Security', 'Database Indexing', 'Container Orchestration', 'Load Balancing', 'Caching Strategies'])[1 + (generate_series % 5)]
        WHEN 17 THEN 'Optimizing ' || (ARRAY['Database Queries', 'API Performance', 'Frontend Loading', 'Memory Usage', 'Network Latency'])[1 + (generate_series % 5)]
        WHEN 18 THEN 'Monitoring and Observability for ' || (ARRAY['Microservices', 'Distributed Systems', 'Database Performance', 'API Health', 'User Experience'])[1 + (generate_series % 5)]
        ELSE 'Tech Trends 2024: ' || (ARRAY['AI Integration', 'Edge Computing', 'Serverless Architecture', 'Low-Code Platforms', 'Quantum Computing'])[1 + (generate_series % 5)]
    END AS title,
    'This is a comprehensive blog post about ' ||
    CASE (generate_series % 10)
        WHEN 0 THEN 'software development best practices and common pitfalls that early-stage startups encounter when building their technology stack. We''ll explore real-world examples and provide actionable insights.'
        WHEN 1 THEN 'system architecture decisions that can make or break your application''s performance. From database design to API architecture, we cover everything you need to know.'
        WHEN 2 THEN 'performance optimization techniques that have helped companies scale from zero to millions of users. Includes code examples and benchmarking results.'
        WHEN 3 THEN 'the technical challenges faced during rapid growth and how to prepare your infrastructure for scale. Based on real startup experiences and lessons learned.'
        WHEN 4 THEN 'modern development practices that improve code quality, team productivity, and system reliability. Includes practical tips and tool recommendations.'
        WHEN 5 THEN 'database optimization strategies that can dramatically improve application performance. We''ll cover indexing, query optimization, and schema design patterns.'
        WHEN 6 THEN 'API design principles that lead to maintainable and scalable systems. Includes examples of good and bad API design with real-world implications.'
        WHEN 7 THEN 'monitoring and observability practices that help teams detect and resolve issues before they impact users. Tools, metrics, and alerting strategies included.'
        WHEN 8 THEN 'security best practices for modern web applications. Covers authentication, authorization, data protection, and common vulnerability prevention.'
        ELSE 'the latest trends in technology and how they impact startup product development. Analysis of adoption patterns and practical implementation advice.'
    END || chr(10) || chr(10) ||
    'Key topics covered:' || chr(10) ||
    '• Implementation strategies and best practices' || chr(10) ||
    '• Common pitfalls and how to avoid them' || chr(10) ||
    '• Performance considerations and optimization' || chr(10) ||
    '• Real-world examples and case studies' || chr(10) ||
    '• Tool recommendations and setup guides' || chr(10) || chr(10) ||
    'Whether you''re a startup founder, developer, or technical leader, this post provides practical insights you can apply immediately to improve your technology stack and development processes.' AS content,
    'A comprehensive guide covering implementation strategies, best practices, and lessons learned from real-world startup experiences.' AS summary,
    1 + (generate_series % 10000) AS author_id,
    (ARRAY['Technology', 'Startups', 'Development', 'DevOps', 'Architecture', 'Performance', 'Security', 'Data', 'Mobile', 'AI/ML'])[1 + (generate_series % 10)] AS category,
    CASE (generate_series % 5)
        WHEN 0 THEN ARRAY['tutorial', 'beginner', 'guide']
        WHEN 1 THEN ARRAY['advanced', 'performance', 'optimization']
        WHEN 2 THEN ARRAY['startup', 'scaling', 'growth']
        WHEN 3 THEN ARRAY['best-practices', 'architecture', 'design']
        ELSE ARRAY['troubleshooting', 'debugging', 'monitoring']
    END AS tags,
    (random() * 10000)::integer AS view_count,
    (random() * 500)::integer AS like_count,
    CASE WHEN generate_series % 20 = 0 THEN false ELSE true END AS published,
    CURRENT_TIMESTAMP - (random() * interval '180 days') AS created_at
FROM generate_series(1, 100000);

-- Generate 500,000 comments with realistic threading
INSERT INTO comments (content, post_id, author_id, parent_comment_id, like_count, created_at)
SELECT
    CASE (generate_series % 15)
        WHEN 0 THEN 'Great article! This really helped me understand the concepts better. Thanks for sharing your experience.'
        WHEN 1 THEN 'I had a similar issue in my project and this approach worked perfectly. Really appreciate the detailed explanation.'
        WHEN 2 THEN 'Excellent deep dive into this topic. The code examples are particularly helpful for implementation.'
        WHEN 3 THEN 'This is exactly what I was looking for! Bookmarking this for future reference. Keep up the great work!'
        WHEN 4 THEN 'Interesting perspective. I''ve been using a different approach but will definitely try this method.'
        WHEN 5 THEN 'Thanks for the comprehensive guide. The step-by-step breakdown makes it easy to follow along.'
        WHEN 6 THEN 'Really insightful post. I learned something new today. Looking forward to more content like this.'
        WHEN 7 THEN 'This solved a problem I''ve been struggling with for weeks. The explanation is clear and concise.'
        WHEN 8 THEN 'Great timing on this article! I was just researching this topic for my current project.'
        WHEN 9 THEN 'Love the practical examples. It''s refreshing to see real-world applications rather than just theory.'
        WHEN 10 THEN 'This is why I follow this blog. Always quality content with actionable insights.'
        WHEN 11 THEN 'Quick question: have you encountered any edge cases with this approach? Overall great article though!'
        WHEN 12 THEN 'Fantastic write-up! The performance implications you mentioned are particularly important for our use case.'
        WHEN 13 THEN 'Well written and informative. I''ll be sharing this with my team for our next architecture review.'
        ELSE 'Thanks for sharing! This gives me some good ideas for optimizing our current implementation.'
    END AS content,
    1 + (generate_series % 100000) AS post_id,
    1 + (generate_series % 10000) AS author_id,
    CASE
        WHEN generate_series % 4 = 0 THEN NULL
        ELSE GREATEST(1, generate_series - 1 - (random() * 100)::integer)
    END AS parent_comment_id,
    (random() * 50)::integer AS like_count,
    CURRENT_TIMESTAMP - (random() * interval '120 days') AS created_at
FROM generate_series(1, 500000);

-- Update statistics for better query planning
ANALYZE users;
ANALYZE posts;
ANALYZE comments;