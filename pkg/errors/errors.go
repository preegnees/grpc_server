package errors

import "fmt"

var ErrInvalidIdChannelWhenRemove error = fmt.Errorf("Ошибка при удалении пира, такого канала не существует")
var ErrInvalidPeerWhenRemove error = fmt.Errorf("Ошибка при удалении пира, такого пира не существует")
var ErrInvalidAllowedNamesWhenSave error = fmt.Errorf("Ошибка при сохранении пира, allowedNames подключающегося клиентя не соответсвует allowedNames клиента в базе")
