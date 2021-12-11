# mdmv - Move Markdown files

Move Markdown files with attachments on filesystem.

`mdmv` helps you reorganize a collection of markdown notes.
It does more than you expect from `mv`, like:

- Supports wildcards to find source files
- Retains correct links to attachments
- Use title from the markdown content as part of the destination path (folder or file name)

### Example

```
mdmv notes/*.md content/%title/README.md

# This command will convert files with this structure 

|-- notes/
    |-- note1.md
    |-- images/
        |-- image1.png

# To this structure
  
|-- content/
    |-- My_personal_note/
        |-- README.md
        |-- images/
            |-- image1.png
```

### Roadmap

- [ ] Support path templating using markdown Frontmatter metadata
- [ ] Specify custom path to attachments as `--attachments` flag
- [ ] Make OS aware modificators for template tags

### Current state

`mdmv` is in its very early stages and welcomes all contributors. 
Ensure that you have a backup of the files before using `mdmv`. Use it on your own risk. 
I've tested it on my personal use-cases, that might not apply for you.
