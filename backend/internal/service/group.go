package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/dto"
	"github.com/benzi/slink/backend/internal/model"
	"github.com/benzi/slink/backend/internal/repo"
	"github.com/benzi/slink/backend/pkg/errcode"
)

type GroupService struct {
	group *repo.GroupRepo
}

func NewGroupService(group *repo.GroupRepo) *GroupService {
	return &GroupService{group: group}
}

func (s *GroupService) List(ctx context.Context) ([]dto.GroupResp, error) {
	groups, err := s.group.List()
	if err != nil {
		return nil, err
	}
	counts, err := s.group.CountLinksByGID()
	if err != nil {
		return nil, err
	}
	out := make([]dto.GroupResp, 0, len(groups))
	for _, g := range groups {
		out = append(out, dto.GroupResp{
			GID:       g.GID,
			Name:      g.Name,
			SortOrder: g.SortOrder,
			LinkCount: counts[g.GID],
			CreatedAt: g.CreatedAt,
		})
	}
	return out, nil
}

func (s *GroupService) Create(ctx context.Context, req dto.GroupReq) (*dto.GroupResp, error) {
	if _, err := s.group.GetByName(req.Name); err == nil {
		return nil, errcode.GroupNameExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	sort := 0
	if req.SortOrder != nil {
		sort = *req.SortOrder
	}
	g := &model.Group{GID: uuid.NewString()[:16], Name: req.Name, SortOrder: sort}
	if err := s.group.Create(g); err != nil {
		return nil, err
	}
	return &dto.GroupResp{GID: g.GID, Name: g.Name, SortOrder: g.SortOrder, CreatedAt: g.CreatedAt}, nil
}

func (s *GroupService) Update(ctx context.Context, gid string, req dto.GroupReq) error {
	g, err := s.group.GetByGID(gid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.GroupNotFound
	}
	if err != nil {
		return err
	}
	// 名称查重（排除自身）
	if same, err := s.group.GetByName(req.Name); err == nil {
		if same.GID != gid {
			return errcode.GroupNameExists
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	g.Name = req.Name
	if req.SortOrder != nil {
		g.SortOrder = *req.SortOrder
	}
	return s.group.Update(g)
}

func (s *GroupService) Delete(ctx context.Context, gid string) error {
	if _, err := s.group.GetByGID(gid); errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.GroupNotFound
	} else if err != nil {
		return err
	}
	n, err := s.group.CountLinks(gid)
	if err != nil {
		return err
	}
	if n > 0 {
		return errcode.GroupHasLinks
	}
	return s.group.Delete(gid)
}

func (s *GroupService) Sort(ctx context.Context, req dto.GroupSortReq) error {
	for _, item := range req.Items {
		g, err := s.group.GetByGID(item.GID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: %s", errcode.GroupNotFound, item.GID)
		}
		if err != nil {
			return err
		}
		g.SortOrder = item.SortOrder
		if err := s.group.Update(g); err != nil {
			return err
		}
	}
	return nil
}
