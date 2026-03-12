# Contributing to StageFlow

First off, thank you for considering contributing to StageFlow! It's people like you that make StageFlow such a great tool.

## Where do I go from here?

If you've noticed a bug or have a feature request, make sure to check our [Issues](https://github.com/mattboback/stageflow/issues) first. If it doesn't exist, go ahead and create a new one!

## Fork & create a branch

If this is something you think you can fix, then fork StageFlow and create a branch with a descriptive name.

## Get the test suite running

Make sure your changes don't break anything. Before submitting a pull request, please ensure you have set up your local environment and run the test suite.

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Run the test suite:
   ```bash
   just ci
   ```

Please make sure all tests pass and your code is properly formatted before submitting a Pull Request.

## Pull Request Process

1. Ensure any install or build dependencies are removed before the end of the layer when doing a build.
2. Update the README.md with details of changes to the interface, if applicable.
3. You may merge the Pull Request in once you have the sign-off of other developers, or if you do not have permission to do that, you may request the reviewer to merge it for you.

## Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms. See `CODE_OF_CONDUCT.md` for more information.
