# Drift Checker

This is a command-line tool that helps you detect **infrastructure drift** between what's deployed in **AWS EC2** and what is defined in your **Terraform configuration** (state file).

It ensures that your live AWS environment matches your Terraform-defined infrastructure — a key part of keeping infrastructure-as-code (IaC) in sync.

---

### 🔍 How It Works

1. **Read Terraform State**  
   Parses a `.tfstate` file to extract expected configuration of EC2 instances.

2. **Query AWS EC2**  
   Calls AWS API (via the SDK) to get the actual configuration of specified EC2 instances.

3. **Compare Fields**: Compares both versions on a field-by-field basis:
   - instance_type
   - AMI ID
   - subnet, VPC
   - tags, security groups
   - block device mappings
   - IAM instance profile, monitoring, and more

4. **Detect Drift**  
   Outputs a structured report highlighting any mismatched fields.

5. **Interactive Fallbacks**  
   If CLI flags are missing, it interactively prompts for:
   - Terraform state path
   - EC2 instance IDs

6. **Formats Output**  
   You can view the drift report in:
   - Human-readable format (default)
   - JSON (with `--json` flag)

---

### Example Use Case

You're managing EC2 instances with Terraform. Someone changes the instance type or tags **manually via the AWS console**.  
→ `drift-checker` will detect that drift so you can fix it.

---

## Features

- Fetches EC2 instance configuration from AWS (via SDK v2)
- Parses Terraform state files (`.tfstate`)
- Compares configurations field-by-field
- Supports:
    - instance type
    - AMI ID
    - tags
    - subnet, security groups
    - block devices, monitoring, architecture
- Concurrent drift detection
- Human-readable and JSON output
- Optional CLI interactivity when flags are missing

---

## 🚀 Installation

```bash
git clone https://github.com/your-username/firefly-assessment.git
cd firefly-assessment
go mod tidy
```

---

## 🔐 AWS Authentication Setup

This tool uses the **AWS SDK for Go v2**, which follows the [default credential chain](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configuring-sdk.html).

You can authenticate by running this in your terminal:

```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=your-aws-region
```

---

## How to run

### ✅ Run with flags (non-interactive)

```bash
go run . \
  --state-file=file/tf.tfstate \
  --instance-ids=id1,id2 \
  --attributes=instance_type,tags \
  --json
```

_Replace `id1,id2` with your own comma seperated list of IDs. When the `-json` flag is added, the results are pasted in
`results` folder with the format `drift_<instance-id>_timestamp.json`_

### ✅ Run interactively (omit flags)
You’ll be prompted to input:
- Path to the Terraform state file
- One or more EC2 instance IDs

```bash
go run .
```
OR
```bash
make run-json
```
OR
```bash
make run
```
---

## Tools & Technologies Used

| Tool / Library                  | Purpose                                   |
|---------------------------------|-------------------------------------------|
| **Go 1.23+**                    | Core programming language                 |
| **AWS SDK for Go v2**           | Fetch EC2 instance config from AWS        |
| **Terraform JSON State Format** | Input source for expected infrastructure  |
| **urfave/cli/v2**               | Command-line flag parsing and UX          |
| **promptui**                    | Interactive CLI prompts for fallback input|
| **stretchr/testify**            | Unit test assertions and mocks            |
| **Go Test & Coverage**          | Built-in Go testing framework             |
| **Make**                        | Automate build, test, lint, and coverage  |
---

## Future Improvements
* Add support for HCL parsing (non-state)
* Compare across multiple resource types
* Export drift reports (CSV, HTML)
* GitHub Actions for test + coverage badge
---

## Requirements Met
1. [x] Go modules
2. [x] Error handling + logging
3. [x] Unit tests
4. [x] Structured reports
5. [x] Concurrency
6. [x] CLI interface
7. [x] README + usage examples