# mdmv - Move Markdown files

Manipulate markdown files with attachments

- [ ] Retains correct links to attachments 
- [ ] Supports path templating using markdown Frontmatter metadata

From this structure:
```
|-- notes/
    |-- note1.md
    |-- images/
        |-- image1.png
```
```
|-- content/
    |-- My_personal_note/
        |-- README.md
        |-- assets/
            |-- image1.png
```

```
mdmv notes content/%title/README.md --attachments content/%title/assets
```
