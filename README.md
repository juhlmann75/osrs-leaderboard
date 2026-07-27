# osrs-leagerboard

Discord bot that posts a player's OSRS Agility score from the hiscores.

## Run

```bash
export BOT_TOKEN=${BOT_TOKEN}
export POST_CHANNEL_ID=${POST_CHANNEL_ID}   # channel for the daily post
export POST_USERNAME=Karwambwadan            # optional, defaults to "Karwambwadan"
export POST_TIME=09:00                       # optional "HH:MM" local time, defaults to 09:00
go run main.go
```

## Behavior

- `!leaderboard [username]` — replies with the agility rank/level/xp for the
  given username (defaults to `POST_USERNAME`).
- Daily scheduled post — once a day at `POST_TIME` (local server time), posts
  the agility score for `POST_USERNAME` to `POST_CHANNEL_ID`. The daily post is
  disabled if `POST_CHANNEL_ID` is unset. The bot does not post on startup; if
  it starts after today's scheduled time, the next post is tomorrow.