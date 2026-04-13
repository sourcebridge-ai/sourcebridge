import { Resend } from "resend";
import { env } from "@/lib/env";

let resendClient: Resend | null = null;

function resend(): Resend {
  if (!resendClient) {
    resendClient = new Resend(env().RESEND_API_KEY);
  }
  return resendClient;
}

export async function sendMagicLinkEmail(
  email: string,
  token: string,
): Promise<void> {
  const url = `${env().NEXT_PUBLIC_APP_URL}/auth/magic-link?token=${token}`;

  await resend().emails.send({
    from: env().EMAIL_FROM,
    to: email,
    subject: "Sign in to Acme",
    html: `
      <h2>Sign in to Acme</h2>
      <p>Click the link below to sign in. This link expires in 15 minutes.</p>
      <a href="${url}" style="
        display: inline-block;
        padding: 12px 24px;
        background: #2563eb;
        color: white;
        text-decoration: none;
        border-radius: 6px;
      ">Sign in</a>
      <p style="color: #6b7280; font-size: 14px; margin-top: 16px;">
        If you didn't request this, you can safely ignore this email.
      </p>
    `,
  });
}

export async function sendInvitationEmail(
  email: string,
  teamName: string,
  token: string,
): Promise<void> {
  const url = `${env().NEXT_PUBLIC_APP_URL}/invite/accept?token=${token}`;

  await resend().emails.send({
    from: env().EMAIL_FROM,
    to: email,
    subject: `You're invited to join ${teamName} on Acme`,
    html: `
      <h2>Team Invitation</h2>
      <p>You've been invited to join <strong>${teamName}</strong> on Acme.</p>
      <a href="${url}" style="
        display: inline-block;
        padding: 12px 24px;
        background: #2563eb;
        color: white;
        text-decoration: none;
        border-radius: 6px;
      ">Accept Invitation</a>
      <p style="color: #6b7280; font-size: 14px; margin-top: 16px;">
        This invitation expires in 7 days.
      </p>
    `,
  });
}

export async function sendWelcomeEmail(
  email: string,
  name: string,
): Promise<void> {
  await resend().emails.send({
    from: env().EMAIL_FROM,
    to: email,
    subject: "Welcome to Acme!",
    html: `
      <h2>Welcome, ${name}!</h2>
      <p>Your account is ready. Here's how to get started:</p>
      <ol>
        <li>Create or join a team</li>
        <li>Invite your colleagues</li>
        <li>Start collaborating</li>
      </ol>
      <a href="${env().NEXT_PUBLIC_APP_URL}/dashboard" style="
        display: inline-block;
        padding: 12px 24px;
        background: #2563eb;
        color: white;
        text-decoration: none;
        border-radius: 6px;
      ">Go to Dashboard</a>
    `,
  });
}
