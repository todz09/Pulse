import 'dotenv/config';
import pkg from 'pg';
import nodemailer from 'nodemailer';
import cron from 'node-cron';

const {Pool} = pkg;

// Create a connection pool to the same database the Go backend writes to.
const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
});

// Configure the email sender using Gmail's SMTP service/
const transporter = nodemailer.createTransport({
    service : 'gmail',
    auth: {
        user : process.env.GMAIL_USER,
        pass : process.env.GMAIL_APP_PASSWORD
    },
});

/**
 * Fetches each monitor along with its most recent check result
 * and its last known alert state.
 */

async function getMonitorStates(){
    const query = `
    SELECT
    m.id,
    m.name,
    m.url,
    c.up AS current_up,
    c.status_code,
    c.error_message,
    c.checked_at,
    a.last_known_up
    FROM monitors m
    LEFT JOIN LATERAL (
    SELECT up, status_code, error_message, checked_at
    FROM checks
    WHERE monitor_id = m.id
    ORDER BY checked_at DESC
    LIMIT 1
    ) c ON true
     LEFT JOIN alert_state a ON a.monitor_id = m.id
     WHERE c.up IS NOT NULL`;

    const result = await pool.query(query);
    return result.rows;
}

/**
 * Records the new state for a monitor so we don't alert twice for the same event.
 */

async function updateAlertState(monitorID, isUp) {
    await pool.query(
        ` INSERT INTO alert_state (monitor_id, last_known_up, last_alerted_at)
        VALUES ( $1, $2, now())
        ON CONFLICT (monitor_id)
        DO UPDATE SET last_known_up = $2, last_alerted_at = now()`,
        [monitorID, isUp]
    );
}

/**
 * Sends an email alert about a monitor changing state.
 */

async function sendAlert(monitor, wentDown){
    const subject = wentDown
    ?`🔴 DOWN : ${monitor.name}`
    :`🟢 UP : ${monitor.name}`;

    const body = wentDown
    ? `${monitor.name} (${monitor.url}) is DOWN.

Status code: ${monitor.status_code ?? 'none'}
Error: ${monitor.error_message ?? 'none'}
Detected at: ${monitor.checked_at}`
    : `${monitor.name} (${monitor.url}) is back UP.

Status code: ${monitor.status_code ?? 'none'}
Recovered at: ${monitor.checked_at}`;

  await transporter.sendMail({
    from: process.env.GMAIL_USER,
    to: process.env.ALERT_TO_EMAIL,
    subject,
    text: body,
  });

  console.log(`Alert sent: ${subject}`);
}

/**
 * The main loop: compare current status against last known status,
 * and alert only when the state has changed.
 */
async function checkForAlerts() {
  console.log(`[${new Date().toISOString()}] Checking for state changes...`);

  try {
    const monitors = await getMonitorStates();

    for (const monitor of monitors) {
      const currentlyUp = monitor.current_up;
      const previouslyUp = monitor.last_known_up;

      // First time we've ever seen this monitor: record state, don't alert.
      if (previouslyUp === null || previouslyUp === undefined) {
        await updateAlertState(monitor.id, currentlyUp);
        console.log(`  Initialised state for ${monitor.name}: ${currentlyUp ? 'UP' : 'DOWN'}`);
        continue;
      }

      // No change: nothing to do.
      if (currentlyUp === previouslyUp) {
        continue;
      }

      // State changed — send an alert and record the new state.
      const wentDown = previouslyUp === true && currentlyUp === false;
      await sendAlert(monitor, wentDown);
      await updateAlertState(monitor.id, currentlyUp);
    }
  } catch (err) {
    console.error('Error while checking for alerts:', err);
  }
}

// Run once immediately on startup.
checkForAlerts();

// Then run every minute.
cron.schedule('* * * * *', checkForAlerts);

console.log('Alert service started. Watching for state changes every minute.');