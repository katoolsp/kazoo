# Kazoo - The Friendly Package Installer

Kazoo is a simple and user-friendly package installer designed for all systems. It allows you to easily install and manage packages from a centralized repository.

## Features

- **Easy Installation**: Quickly install packages with a single command.
- **Package Management**: Install and remove packages effortlessly.
- **Version Control**: Specify package versions for installation.
- **Cross-Platform Support**: Written in Go, Kazoo works with all architectures.

## Installation

To install Kazoo, follow these steps:

1. Clone the repository:
   ```bash
   git clone https://github.com/katoolsp/kazoo.git
   cd kazoo
   ```
2. Build the project:
   ```bash
   make
   ```
3. Install Kazoo:
   ```bash
   make install
   ```
This will move the `kazoo` binary to `/usr/local/bin/` and make it executable.

## Usage
Once installed, you can use Kazoo from the command line. Here are some common commands:
- Check Version:
  ```bash
  kazoo
  ```
  > !!Notice!!: It is highly recommended to run all install, remove, and update commmands in Kazoo with super-user privilege to avoid permission and package list issues.
- Install a Package:
  ```bash
  sudo kazoo -i <e.g. hello, neofetch>
  ```
- Remove a Package:
  ```bash
  sudo kazoo -r <e.g. hello, neofetch>
  ```
- Update a Package:
  ```bash
  sudo kazoo -u <e.g. hello, neofetch or blank to update Kazoo>
  ```

## Contributing
Contributions are welcome! If you have suggestions or improvements, feel free to open an issue or submit a pull request.

## License
Kazoo is licensed under the Free and Fair License. See the LICENSE file for details.

Happy packaging with Kazoo!
