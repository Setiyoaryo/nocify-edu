"""Content intelligence pipeline for Nocify EduPlay.

This service owns the ``content_staging`` schema. It never writes canonical
content: approved candidates are handed to the Go API through an explicit
publish contract (see docs/content-pipeline-roadmap.md).
"""

__version__ = "0.1.0"
