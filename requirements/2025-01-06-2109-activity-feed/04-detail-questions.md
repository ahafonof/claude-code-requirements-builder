# Expert Detail Questions

## Q6: Чи потрібно емітувати подію активності одразу в RateLimitMiddleware при відхиленні запиту (до відправки 429 відповіді)?
**Default if unknown:** Yes (негайна емісія забезпечує точний порядок подій)

## Q7: Чи повинна веб-сторінка для перегляду стрічки активності бути вбудована в main.go як HTML-рядок (без окремого файлу)?
**Default if unknown:** Yes (простіше розгортання, один бінарний файл містить все)

## Q8: Чи потрібно обмежити кількість одночасних SSE-клієнтів для activity feed (наприклад, максимум 100)?
**Default if unknown:** Yes (захист від вичерпання ресурсів сервера)

## Q9: Чи повинні події в ActivityFeed мати унікальні ID у форматі timestamp-sequence для точного упорядкування?
**Default if unknown:** Yes (гарантує унікальність та правильний порядок подій)

## Q10: Чи потрібно додати поле "severity" до ActivityEvent для візуального виділення критичних подій (error/warning/info)?
**Default if unknown:** No (всі важливі події мають однакову вагу, користувач не просив категоризації)