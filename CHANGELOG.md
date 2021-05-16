# Changelog

## Unversioned

- Minor: Moderator commands now also check for server ownership in addition to moderator role
- Minor: Add ability to configure various uncategorized values using the `$configure value` command. Use `$configure value keys` to see a list of valid keys and their description. (#46)
- Minor: The \$points command now has a configurable pajbot host instead of pointing at `forsen.tv`. Set it with `$configure value set pajbot_host forsen.tv`. (#46)
- Minor: Added \$8ball command
- Bugfix: Markdown escaping did not escape \` and handle backslashes properly
