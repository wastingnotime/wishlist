# Admin derived slugs

The admin browser now derives an app slug from its name and a feature slug from its title when creating them. Suggestion acceptance uses the same conversion. The conversion removes accents, replaces punctuation and spaces with hyphens, and limits slugs to the API's 80-character maximum. Names and titles that cannot produce an alphanumeric slug show a form message.

The API and simulation continue to require an explicit slug in their create commands. Existing slugs remain stable when an app name or feature title is edited.

Validation: `npm run typecheck` passed; the Playwright admin create-and-publish flow passed with an accented app name and a punctuated feature title.
