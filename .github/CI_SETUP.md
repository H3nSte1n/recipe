# GitHub CI Setup

Simple CI pipeline for the backend.

## What it does

✅ Runs tests with PostgreSQL  
✅ Builds the Go binary  
✅ Only runs when backend code changes  

## When it runs

- Push to `main` or `develop` branch
- Pull requests to `main` or `develop`
- Only if files in `services/backend/` changed

## Test it

1. Make a change to any backend file
2. Commit and push to GitHub
3. Go to **Actions** tab
4. Watch the pipeline run

## Troubleshooting

**Tests fail?**
- Make sure your code compiles locally: `go build ./...`
- Run tests locally: `go test ./...`

**Badge not showing?**
- Replace `yourusername` in README badge with your GitHub username

That's it! Keep it simple. ✅
