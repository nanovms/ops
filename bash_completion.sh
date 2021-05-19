#!/bin/sh

__nanovms_ops_completions() {
    if [ "${#COMP_WORDS[@]}" = "2" ]; then
        COMPREPLY+=($(compgen -W "help build deploy image instance pkg profile run update version volume" "${COMP_WORDS[1]}"))
        return
    fi

     if [ "${#COMP_WORDS[@]}" = "3" ]; then
        case "${COMP_WORDS[1]}" in
            pkg)
                COMPREPLY+=($(compgen -W "contents describe get list load" "${COMP_WORDS[2]}"))
                ;;
            instance)
                COMPREPLY+=($(compgen -W "create delete list logs start stop" "${COMP_WORDS[2]}"))
                ;;
            image)
                COMPREPLY+=($(compgen -W "create delete list resize sync" "${COMP_WORDS[2]}"))
                ;;
            volume)
                COMPREPLY+=($(compgen -W "attach create delete detach list" "${COMP_WORDS[2]}"))
                ;;    
            *)
        esac
        return
     fi
}

complete -F __nanovms_ops_completions ops