CREATE TABLE messages (
    id              SERIAL       PRIMARY KEY,
    recipient       VARCHAR(20)  NOT NULL,
    content         VARCHAR(160) NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    sent_at         TIMESTAMP WITH TIME ZONE,
    retry_count     INT          NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE,
    response_id     VARCHAR(255),
    response_code   INT,
    error_message   TEXT
);



