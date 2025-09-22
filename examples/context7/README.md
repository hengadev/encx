# Context7 Examples for Encx

This directory contains structured examples specifically organized for Context7 users, categorized by use case and complexity level.

## Directory Structure

```
context7/
├── README.md                          # This file
├── 01-basic/                          # Beginner examples
│   ├── simple_encryption.go          # Basic field encryption
│   ├── basic_hashing.go              # Simple hashing for lookups
│   └── password_security.go          # Password hashing
├── 02-intermediate/                   # Real-world patterns
│   ├── user_management.go            # Complete user system
│   ├── searchable_encryption.go      # Encrypt + search capability
│   └── database_integration.go       # Database operations
├── 03-advanced/                       # Production patterns
│   ├── code_generation.go            # High-performance approach
│   ├── key_rotation.go               # Key management
│   └── multi_tenant.go               # Multi-tenant encryption
└── 04-industry/                       # Industry-specific examples
    ├── healthcare_pii.go             # HIPAA compliance
    ├── financial_data.go             # Financial data protection
    └── ecommerce_customer.go         # E-commerce customer data
```

## Quick Navigation by Use Case

### 🔐 Data Protection
- **Personal Information**: `01-basic/simple_encryption.go`
- **Healthcare Data**: `04-industry/healthcare_pii.go`
- **Financial Records**: `04-industry/financial_data.go`

### 🔍 Search & Lookup
- **User Search**: `02-intermediate/searchable_encryption.go`
- **Customer Lookup**: `04-industry/ecommerce_customer.go`
- **Database Queries**: `02-intermediate/database_integration.go`

### 🔑 Authentication
- **Password Security**: `01-basic/password_security.go`
- **User Management**: `02-intermediate/user_management.go`

### ⚡ Performance
- **Code Generation**: `03-advanced/code_generation.go`
- **Batch Processing**: `02-intermediate/database_integration.go`

### 🏢 Enterprise
- **Key Rotation**: `03-advanced/key_rotation.go`
- **Multi-tenant**: `03-advanced/multi_tenant.go`

## Example Complexity Levels

| Level | Description | When to Use |
|-------|-------------|-------------|
| **Basic** | Single concept examples | Learning individual features |
| **Intermediate** | Real-world patterns | Building production features |
| **Advanced** | Optimization techniques | Performance/scale requirements |
| **Industry** | Compliance-focused | Specific regulatory needs |

## Getting Started

1. **New to Encx?** Start with `01-basic/simple_encryption.go`
2. **Building user system?** See `02-intermediate/user_management.go`
3. **Need performance?** Check `03-advanced/code_generation.go`
4. **Compliance requirements?** Browse `04-industry/` directory

Each example includes:
- ✅ Complete, runnable code
- 📝 Detailed comments explaining concepts
- 🧪 Test cases demonstrating usage
- 📊 Performance considerations
- 🔒 Security best practices

## Context7 Search Patterns

Use these queries to find relevant examples:

- **"golang encrypt struct field"** → `01-basic/simple_encryption.go`
- **"searchable encryption golang"** → `02-intermediate/searchable_encryption.go`
- **"password hashing best practices"** → `01-basic/password_security.go`
- **"golang encryption performance"** → `03-advanced/code_generation.go`
- **"HIPAA compliant encryption"** → `04-industry/healthcare_pii.go`
- **"database encryption golang"** → `02-intermediate/database_integration.go`