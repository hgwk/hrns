#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { exists, readJson } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-task-workflow')
const config = configSection('workflow')
const mode = config.mode ?? 'warn'
const todoPath = config.todoPath ?? 'tasks/todo.json'
const lessonsPath = config.lessonsPath ?? 'tasks/lessons.json'
const findings = []

if (!exists(todoPath)) {
  findings.push({ message: `${todoPath} is missing` })
} else {
  const todo = readJson(todoPath)
  if (!Array.isArray(todo.items) || todo.items.length === 0) {
    findings.push({ message: `${todoPath} should contain a non-empty items array` })
  }
  if (!todo.verification || typeof todo.verification !== 'object') {
    findings.push({ message: `${todoPath} should contain a verification object` })
  }
  if (!todo.review || typeof todo.review !== 'object') {
    findings.push({ message: `${todoPath} should contain a review object` })
  }
  for (const [index, item] of (todo.items ?? []).entries()) {
    if (typeof item.task !== 'string' || !item.task.trim()) {
      findings.push({ message: `${todoPath} item missing task`, detail: `items[${index}]` })
    }
    if (!['todo', 'doing', 'done', 'blocked'].includes(item.status)) {
      findings.push({ message: `${todoPath} item has invalid status`, detail: `items[${index}].status` })
    }
  }
}

if (!exists(lessonsPath)) {
  findings.push({ message: `${lessonsPath} is missing` })
} else {
  const lessons = readJson(lessonsPath)
  if (!Array.isArray(lessons.lessons)) {
    findings.push({ message: `${lessonsPath} should contain a lessons array` })
  }
}

finishByMode(audit, findings, mode)
