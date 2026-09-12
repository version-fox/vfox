# Fish completion uses the same command definitions as the vfox CLI.
function __vfox_fish_complete
    set -l args (commandline -opc)
    set -l current (commandline -ct)

    # The CLI disables completion after -- and would execute the command.
    if contains -- -- $args
        return
    end

    if string match -q -- '-*' "$current"
        # Ask for all flags and let Fish filter the prefix.
        # Never pass a literal --, which would execute the command instead.
        set -a args -
    end

    # A parent Zsh may have left SHELL set to zsh. Request plain candidates.
    set -lx SHELL fish
    command $args --generate-shell-completion 2>/dev/null
end

complete -c vfox -f -a '(__vfox_fish_complete)'
