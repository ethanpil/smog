# Smog - TODO List

This file tracks proposed features and enhancements for future development.

## Preserve Bcc Privacy by Sending Separate Emails

**Goal:** Implement true `Bcc` (Blind Carbon Copy) functionality, where `Bcc` recipients receive the email without their addresses being visible to anyone else.

**Problem:** The current implementation ensures `Bcc` recipients receive the email, but it does so by adding them to the public `To:` header, which makes their email address visible to all other recipients. This is a necessary workaround due to the constraints of sending raw email data via the Gmail API.

**Proposed Solution (Option 1):**

Instead of sending one email to everyone, the application would send the message in batches:

1.  The application would first send **one** email to all the public recipients (everyone in the original `To:` and `Cc:` fields). The `To:` and `Cc:` headers for this email would look correct to these recipients.
2.  Then, the application would loop through each `Bcc` recipient and send a **separate, individual email** to each one. For these deliveries, the `To:` header would only contain that single `Bcc` recipient's address.

**Pros:**

*   **Preserves Bcc Privacy:** The "blind" aspect is fully restored. No recipient will see the addresses of the `Bcc` users.
*   **Maintains Message Integrity:** The original email body and attachments remain untouched.

**Cons:**

*   **Inefficient:** This approach increases the number of API calls. If you send an email with 5 `Bcc` recipients, it would result in 6 separate `Send` calls to Google (1 for the main thread + 5 for each `Bcc`).
*   **Rate Limiting:** This could cause the application to hit Gmail's API rate limits more quickly if emails with many `Bcc` recipients are sent frequently.
