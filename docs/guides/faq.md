# FAQ

## Switch xxx not work or the vfox use  command does not work ?

If your shell prompt `Warning: The current shell lacks hook support or configuration. It has switched to global scope
automatically` that means you do not hook `vfox` into your shell, please hook it manually first.

Please refer to [Quick Start#_2-hook-vfox-to-your-shell](./quick-start.md#_2-hook-vfox-to-your-shell) to manually hook `vfox` into your shell.

## Why can't I delete the plugin?

```text
I first added java/adoptium-jdk, then tried to install v21, because the download was slow, I exited halfway, and then tried
to remove the plugin with the remove command, and got the error message "java/adoptium-jdk not installed".

So I want to switch to another source, when I execute "vfox add java/azul-jdk", I also get the error message "plugin java
already exists", now I can't move forward or backward.

```

In the `vfox` concept, the plugin is the SDK, and the SDK is the plugin. You can think of the plugin as an extension of `vfox`
to manage different tools and runtime environments.

Taking the `nodejs/npmmirror` plugin as an example, `nodejs` is the category, `npmmirror` is the plugin name, and the
**SDK name** marked by the `name` field inside the plugin.

So, when deleting the plugin, you need to use the **SDK name** (here is `nodejs`) for deletion, not the plugin name
`nodejs/npmirror` or `npmmirror`.

```bash
$ vfox remove nodejs
```

before deleting, you can use `vfox ls` to view the currently installed plugins (i.e. SDK names), and then delete them.