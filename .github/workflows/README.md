# GitHub Workflows

This directory contains automated workflows for the encx project.

## Context7 Integration (`context7.yml`)

Automatically updates the encx library documentation on the Context7 platform.

### Triggers

- **Releases**: Automatically adds/updates documentation when a new release is published
- **Documentation changes**: Updates Context7 when docs are modified on main branch
- **Manual trigger**: Can be run manually with custom operation type

### Operations

- **Add**: Submits the library to Context7 for the first time
- **Refresh**: Updates existing library documentation

### Usage

#### Automatic (Recommended)
The workflow runs automatically when:
- A new release is published
- Documentation files are updated on main branch

#### Manual Trigger
1. Go to the **Actions** tab in GitHub
2. Select **"Update Context7 Documentation"**
3. Click **"Run workflow"**
4. Choose operation type:
   - `add` - For first-time submission
   - `refresh` - For updating existing documentation

### Files Monitored
- `docs/**` - All documentation files
- `README.md` - Main repository documentation
- `examples/**` - All example files
- `CONTEXT7_METADATA.md` - Context7 library metadata
- `doc.go` - Package documentation
- `*.md` - All markdown files

### Validation

The workflow automatically validates:
- ✅ Required Context7 files exist
- ✅ Examples directory structure is correct
- ✅ Go example files compile without errors
- ✅ Documentation contains Context7 references

### Troubleshooting

If the workflow fails:
1. Check the workflow logs in GitHub Actions
2. Verify all required files are present
3. Ensure repository is public
4. Try manual trigger with `add` operation
5. Check Context7 service status

### Expected Results

After successful execution:
- Library will be searchable on Context7.com
- Documentation will be available through Context7 MCP
- Users can find encx patterns and examples via Context7
- Automatic updates on future releases/documentation changes