#!/bin/bash
main() {
  curl -X POST -d '{"username":"kaede","password":"kaede"}' -H "Content-Type: application/json" http://localhost:8080/api/v1
}

# Run main function.
main
