# Your own cloud server (optional)

By default Desk Buddy runs on your Mac. That's all most people need.

*Why it exists at all:* honestly, it's a bit overkill. I added it because my Kindle kept dropping its USB connection whenever I copied anything big to it, so I needed a way to send it new versions over the internet instead. Running the server on my own Cloudflare account did that, and it stuck around because it's handy.

A cloud server is worth it if you want:

- the Kindle to keep working while your Mac sleeps or is off,
- the phone page to work away from home,
- two computers (say a Mac and a Windows PC) without one depending on the other.

It runs in **your own** free Cloudflare account (Workers and D1 on the free plan, which is plenty). Nobody else's server is involved, and only someone with your key can read it.

## The easy way: from the Mac app

Desk Buddy › **Server** › **Your own cloud server** walks you through it:

1. **Make a free Cloudflare account** (the button opens the sign-up page).
2. **Install Node.js** (the LTS version from [nodejs.org](https://nodejs.org)); the setup needs it once. The app shows ✓ when it's there.
3. **Run in Terminal**: Terminal opens and does the rest. Your browser opens once to log in to Cloudflare. At the end your server's address (`https://desk-buddy.<you>.workers.dev`) is printed and copied.
4. **Paste the address** and click **Connect**. Your tasks are copied over, Desk Buddy switches to the cloud, and the Kindle follows by itself.

Then scan the **phone page** QR code again (Server page), because the address changed.

### What runs

The app shows the exact commands before you run them. In short: it copies the server's code and a small `setup.sh` into `~/Library/Application Support/DeskBuddy/cloud-setup`, with your key in a private file next to it, and runs:

```sh
cd ~/Library/Application\ Support/DeskBuddy/cloud-setup
sh setup.sh
```

which uses Cloudflare's own tool, Wrangler:

```sh
npm install
npx wrangler login
npx wrangler d1 create desk-buddy
npx wrangler d1 execute desk-buddy --remote --file schema.sql
npx wrangler deploy
npx wrangler secret put KEY
```

You can read the whole script with **Open the script**, or in [`worker/setup.sh`](../worker/setup.sh).

## Let an AI do it

**Copy prompt** (same page) copies instructions you can paste into **Claude Code** or **Codex**: they can run every step for you. ChatGPT or Claude in the browser can't run anything on your Mac, but the prompt asks them to walk you through it step by step. Your key is never in the prompt.

## Coming: a Deploy to Cloudflare button

A one-click button here, with no Node.js and no Terminal. It needs a GitHub account (it copies this project to your account and deploys it from there).

## Switching back

Desk Buddy › Server › **This Mac** (copy your tasks back when it asks). The Kindle follows. Your cloud server stays in your Cloudflare account until you delete it there (Workers & Pages › desk-buddy, and D1 › desk-buddy).
