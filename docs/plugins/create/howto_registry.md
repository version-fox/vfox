# How to publish a plugin to the registry?

::: tip

Please note that this document is about how to submit a plugin to
the [registry](https://github.com/version-fox/vfox-plugins).

If you want to know how to create a plugin, please check [here](./howto.md).
:::

## Registry

The `vfox` plugin registry is a repository used to **collect and distribute** various **vfox plugins**.
It allows users to quickly install the corresponding plugins by running `vfox add <plugin-name>`.

The registry is a public repository, and anyone can submit **vfox plugins** to this repository.

The registry is mainly divided into two parts:

- `plugins`: Used to store the `manifest.json` file of the plugin, with the plugin short name as the file name. For
  example, `nodejs.json`
- `sources`: Used to store the source information of the plugin manifest, with the plugin short name as the file name.
  For example, `nodejs.json`

The repository will automatically retrieve the latest version information of the plugin and verify
the availability of the plugin through the information in `sources` at regular intervals (every hour),
and store the obtained `manifest` information in the `plugins` directory.

::: tip Registry address
`vfox` will default to retrieve plugins from [plugins registry](https://version-fox.github.io/vfox-plugins).

If you want to use **your own registry or third-party mirror registry**, please configure it according to
the [plugin registry address](../../guides/configuration.md#plugin-registry-address).
:::

## Submit a plugin

1. First, you need to create a plugin according to the [plugin creation guide](./howto.md).
2. Maintain a `manifest.json` file to describe the latest version information of the plugin. (If developed based
   on [vfox-plugin-template](https://github.com/version-fox/vfox-plugin-template), it will be automatically generated
   when released)
    ```json
    {
      "downloadUrl": "https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip",
      "notes": [],
      "version": "0.0.5",
      "homepage": "https://github.com/version-fox/vfox-nodejs",
      "minRuntimeVersion": "0.2.6",
      "license": "Apache 2.0",
      "description": "Node.js runtime environment.",
      "name": "nodejs"
    }
    ```
    - `downloadUrl`: The download address of the plugin
    - `notes`: The update log of the plugin
    - `version`: The version of the plugin
    - `homepage`: The homepage of the plugin
    - `minRuntimeVersion`: The minimum `vfox` version required by the plugin
    - `license`: The license of the plugin
    - `description`: The description of the plugin
    - `name`: The short name of the plugin
3. Create a file with the short name you want `vfox` to use in `sources/<name>.json`, for example, `sources/nodejs.json`
4. Add the plugin's manifest address information in `sources/<name>.json`, for example:
    ```json
    {
      "name": "nodejs",
      "manifestUrl": "https://github.com/version-fox/vfox-nodejs/releases/download/manifest/manifest.json",
      "test": {
        "version": "21.7.1",
        "check": "node -v",
        "resultRegx": "v21.7.1"
      }
    }
    ```
    - `name`: The short name of the plugin
    - `manifestUrl`: The manifest address of the plugin
    - `test`: The test information of the plugin
        - `version`: The version of the plugin
        - `check`: The test command
        - `resultRegx`: The test result regular expression
5. Finally, submit a PR to the [registry](https://github.com/version-fox/vfox-plugins/)
6. After the PR is merged, the plugin will be automatically added to the public registry, and the plugin update status
   will be checked every hour.



