Feature: Generate shell completion without an installation
  The user can obtain shell integration and static suggestions without setup or network access.

  # Source: ../../docs/commands/completion.md#usage
  Rule: Supported shell scripts are generated without setup
    Scenario Outline: The user generates a completion script
      Given the user has no modlist or lockfile
      And Mojang, Modrinth and CurseForge cannot receive requests
      When the user generates completion for <shell>
      Then they should receive a nonempty <shell> completion script
      And they should not be offered initialization
      And Mojang, Modrinth and CurseForge should receive no requests
      And they should find no installation metadata or mod files created

      Examples:
        | shell      |
        | bash       |
        | zsh        |
        | fish       |
        | powershell |

  # Source: ../../docs/commands/completion.md#options
  Rule: Completion descriptions can be omitted explicitly
    Scenario: The user generates completion without descriptions
      Given the user has no modlist or lockfile
      When the user generates fish completion without descriptions
      Then they should receive a nonempty fish completion script
      And the script should omit completion descriptions
      And they should find no installation metadata or mod files created

  # Source: ../../docs/commands/completion.md#suggestions-and-setup
  Rule: Supported command, flag and init value suggestions are static
    Scenario: The user receives command and flag suggestions without setup
      Given the user has no modlist or lockfile
      And Mojang, Modrinth and CurseForge cannot receive requests
      When the user requests completion suggestions for the MMM root command
      Then they should see change among the supported commands
      When the user requests completion suggestions for change options
      Then they should see --force among the supported flags
      And Mojang, Modrinth and CurseForge should receive no requests
      And they should find no installation metadata or mod files created

    Scenario Outline: The user requests supported init values without network access
      Given the user has no modlist or lockfile
      And Mojang, Modrinth and CurseForge cannot receive requests
      When the user requests completion suggestions for <option>
      Then they should see <values> suggested
      And they should not be offered initialization
      And Mojang, Modrinth and CurseForge should receive no requests
      And they should find no installation metadata or mod files created

      Examples:
        | option          | values                       |
        | loader          | fabric, quilt and forge      |
        | release types   | alpha, beta and release      |

    Scenario: Optional completion registration failure does not prevent initialization
      Given the user has an existing usable mods directory
      And Mojang provides a valid Minecraft version manifest containing 1.20.2
      And optional completion registration fails
      When the user initializes with Fabric, Minecraft 1.20.2, a release-only policy, and the existing mods directory
      Then they should find the accepted setup in the modlist
      And they should find an empty lockfile
      And they should be told that initialization completed
