package api

import "net/http"

func legalTermsOfService(w http.ResponseWriter, r *http.Request) {
	const text = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Forsen Discord - Terms of Service</title>
</head>
<body>
  <h1>Forsen Discord - Terms of Service</h1>
  <p>
    Anyone may join Forsen's Discord server assuming you have the invite link (barring those that have been banned).
  </p>
  <p>
    Please read the rules channel, and report any rule-breaking behaviour to moderators.
  </p>
  <p>
    If you have any questions about these Terms of Service, you can <a href="https://github.com/pajbot/pajbot2-discord/issues/new">contact us</a>.
  </p>
  <p>
    <a href="privacy-policy">Our Privacy Policy</a>
  </p>
</body>
</html>`
	w.Write([]byte(text))
}

func legalPrivacyPolicy(w http.ResponseWriter, r *http.Request) {
	const text = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Forsen Discord - Privacy Policy</title>
</head>
<body>
  <h1>Forsen Discord -  Privacy Policy</h1>
  <p>
    We process or store the following information about you and the content you post in the server for moderation purposes:
  </p>
  <ul>
    <li>Your Discord & Twitch User ID - This is what we use for user verification, ensuring everyone can have a safe experience in the Discord server.</li>
    <li>Messages you posted - This helps us run moderation tools more advanced than Discord permits.</li>
    <li>Attachment you upload for ~15 minutes - This helps us ensure people who post rule-breaking attachments can be punished even if they quickly delete their message containing the attachment.</li>
	<li>Your IP address - The authentication system will process you IP address, as every website does (including this one!), but we never store it.</li>
  </ul>
  <p>
    We do not share any information with any internal or external parties.
  </p>
  <p>
    If you have any questions about this Privacy Policy, you can <a href="https://github.com/pajbot/pajbot2-discord/issues/new">contact us</a>.
  </p>
  <p>
    If you wish to opt-out or receive a copy of your data, please send a Discord DM to pajlada and I will respond within 48 hours. I intend to make the opt-out process automated when time allows.
  </p>
  <p>
    <a href="terms-of-service">Our Terms of Service</a>
  </p>
</body>
</html>`
	w.Write([]byte(text))
}

func legalAttach(handler *http.ServeMux) {
	handler.HandleFunc("/legal/terms-of-service", legalTermsOfService)
	handler.HandleFunc("/legal/privacy-policy", legalPrivacyPolicy)
}
