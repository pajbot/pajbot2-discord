# Changelog

## 2023-04-10

- Add basic support for slash commands. (#81)
- Fix: $roll is now 1-"indexed". (#139)

## Undated

- Minor: Moderator commands now also check for server ownership in addition to moderator role
- Minor: Add ability to configure various uncategorized values using the `$configure value` command. Use `$configure value keys` to see a list of valid keys and their description. (#46)
- Minor: The \$points command now has a configurable pajbot host instead of pointing at `forsen.tv`. Set it with `$configure value set pajbot_host forsen.tv`. (#46)
- Minor: Added $8ball command
- Minor: Make $colors command output colorful
- Minor: Add $roll command
- Dev: Bumped minimum Go version to 1.19. (#111)
- Dev: Bumped minimum Go version to 1.18. (#106)
- Dev: Bumped minimum Go version to 1.17. (#97)
