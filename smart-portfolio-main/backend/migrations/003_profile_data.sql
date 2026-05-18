-- Smart Portfolio: Profile Data Schema and Seed
-- This migration creates structured tables for all resume sections and seeds them.

-- =============================================================================
-- Profile / Personal Info
-- =============================================================================
CREATE TABLE IF NOT EXISTS profile (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name     VARCHAR(100) NOT NULL,
    last_name      VARCHAR(100),
    primary_role   VARCHAR(100),
    specialization VARCHAR(100),
    location       VARCHAR(255),
    summary        TEXT,
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- Education
-- =============================================================================
CREATE TABLE IF NOT EXISTS education (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution VARCHAR(255) NOT NULL,
    degree      VARCHAR(255) NOT NULL,
    location    VARCHAR(255),
    start_date  VARCHAR(50),
    end_date    VARCHAR(50),
    gpa         VARCHAR(20),
    coursework  TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- Work Experience
-- =============================================================================
CREATE TABLE IF NOT EXISTS experience (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company     VARCHAR(255) NOT NULL,
    role        VARCHAR(255) NOT NULL,
    location    VARCHAR(255),
    start_date  VARCHAR(50),
    end_date    VARCHAR(50),
    summary     TEXT,
    tech_stack  VARCHAR(512), -- Comma separated
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- Certifications
-- =============================================================================
CREATE TABLE IF NOT EXISTS certifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    issuer     VARCHAR(255) NOT NULL,
    issue_date VARCHAR(50),
    url        VARCHAR(555),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- Achievements
-- =============================================================================
CREATE TABLE IF NOT EXISTS achievements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(255) NOT NULL,
    metric      VARCHAR(100),
    description TEXT,
    date        VARCHAR(50),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- Skills
-- =============================================================================
CREATE TABLE IF NOT EXISTS skills (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category   VARCHAR(100) NOT NULL, -- e.g., 'Languages', 'Backend'
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- SEED DATA
-- =============================================================================

-- Profile
INSERT INTO profile (first_name, last_name, primary_role, specialization, location, summary)
VALUES ('Pranjal', 'Kumar', 'FULLSTACK_DEV', 'FULL_STACK_AI_RAG', 'Haridwar, Uttarakhand', 
'Full-stack programmer focused on building high-performance scalable systems with modern web technologies and AI integration.')
ON CONFLICT DO NOTHING;

-- Education
INSERT INTO education (institution, degree, location, start_date, end_date, gpa, coursework)
VALUES ('Gurukula Kangri (Deemed to be University)', 'B.Tech, Computer Science and Engineering', 'Haridwar, Uttarakhand', 'Aug 2024', 'Present', '9.1 / 10.0', 
'Data Structures & Algorithms, OOP, Discrete Mathematics, Web Development, Database Management Systems, Operating Systems, Computer Networks')
ON CONFLICT DO NOTHING;

-- Experience
INSERT INTO experience (company, role, location, start_date, end_date, summary, tech_stack)
VALUES ('VLED — NPTEL Vinternship', 'MERN Stack Developer Intern', 'IIT Ropar, Punjab (Remote)', 'Jan 2026', 'Mar 2026', 
'Contributed to open-source internship platforms, patched security vulnerabilities, and built RAG-powered AI tools for social impact.', 
'React, Node.js, MongoDB, Gemini AI')
ON CONFLICT DO NOTHING;

-- Projects (Manual ones not in GitHub)
INSERT INTO projects (title, description, tech_stack, github_url, live_url)
VALUES 
('Niti-Setu - AI Government Scheme Discovery', 
'RAG-powered platform for personalized government scheme discovery and eligibility verification for Indian farmers. Implemented a chat assistant using Gemini + LangChain to parse 50-page policy PDFs.', 
'React 19, TypeScript, Node.js, Express, MongoDB, Google Gemini, LangChain', 
'https://github.com/LibreTurtle/niti-setu', 
'https://nitisetu-ajrasakha.vercel.app/'),
('Vi-Notes - Authenticity Verification Platform', 
'Behavioral analysis engine that captures keystroke timing metadata to distinguish human-authored content from AI-generated text. Generates exportable PDF authenticity certificates.', 
'React 19, TypeScript, Node.js, Express, MongoDB, JWT, KaTeX', 
'https://github.com/ZRishu/vi-notes', 
'https://vi-notes-zr.vercel.app/')
ON CONFLICT DO NOTHING;

-- Certifications
INSERT INTO certifications (name, issuer, issue_date, url)
VALUES 
('CS50x: Introduction to Computer Science', 'Harvard University / edX', 'Dec 2025', 'https://cs50.harvard.edu/certificates/30b59868-a373-40c2-9258-e1e95925bb23'),
('Spring Boot 3, Spring 6 & Hibernate for Beginners', 'Udemy', 'June 2025', 'https://ude.my/UC-608a60ac-4e84-4b25-adc3-5f6aa7ebe8b0')
ON CONFLICT DO NOTHING;

-- Achievements
INSERT INTO achievements (title, metric, description, date)
VALUES 
('Ajrasakha Hackathon - Team AjraX', 'BEST_PROJECT', 'Recognized as one of the best projects for real-world impact on rural farmer welfare.', 'Feb 2026'),
('Open Source Contributions', 'VULN_PATCHES', 'Identified and patched watch-time bypass and rendering bugs in VLED platforms.', 'Jan-May 2026')
ON CONFLICT DO NOTHING;

-- Skills
INSERT INTO skills (category, name) VALUES 
('Languages', 'Java'), ('Languages', 'C'), ('Languages', 'C++'), ('Languages', 'Go'), ('Languages', 'JavaScript'), ('Languages', 'TypeScript'), ('Languages', 'Kotlin'), ('Languages', 'Python'),
('Frontend', 'React'), ('Frontend', 'Astro'), ('Frontend', 'Tailwind CSS'),
('Backend', 'Spring Boot'), ('Backend', 'Node.js'), ('Backend', 'Express.js'),
('Databases', 'PostgreSQL'), ('Databases', 'MongoDB'), ('Databases', 'Elasticsearch'),
('Auth & Security', 'Keycloak'), ('Auth & Security', 'JWT'),
('AI / ML', 'Gemini AI'), ('AI / ML', 'LangChain'), ('AI / ML', 'RAG'),
('DevOps', 'Docker'), ('DevOps', 'Git')
ON CONFLICT DO NOTHING;
