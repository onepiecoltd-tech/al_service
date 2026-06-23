-- +goose Up
CREATE TABLE pronunciation_words (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word       TEXT NOT NULL,
    phonetic   TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO pronunciation_words (word, phonetic) VALUES
    ('entrepreneur', '/ˌɒn.trə.prəˈnɜːr/'),
    ('comfortable', '/ˈkʌm.fə.tə.bəl/'),
    ('vegetable', '/ˈvedʒ.tə.bəl/'),
    ('February', '/ˈfeb.ru.ər.i/'),
    ('schedule', '/ˈʃedʒ.uːl/'),
    ('colleague', '/ˈkɒl.iːɡ/'),
    ('rural', '/ˈrʊə.rəl/'),
    ('thorough', '/ˈθʌr.ə/'),
    ('phenomenon', '/fəˈnɒm.ɪ.nən/'),
    ('hierarchy', '/ˈhaɪ.ə.rɑː.ki/'),
    ('queue', '/kjuː/'),
    ('worcestershire', '/ˈwʊs.tə.ʃər/'),
    ('anemone', '/əˈnem.ə.ni/'),
    ('squirrel', '/ˈskwɪr.əl/'),
    ('choir', '/ˈkwaɪər/');

-- +goose Down
DROP TABLE pronunciation_words;
