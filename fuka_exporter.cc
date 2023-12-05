#include <functional>
#include <string>
#include <vector>
#include <array>
#include <type_traits>
#include <iostream>
#include <omp.h>

#include "bin_ns.hpp"
#include "bin_bh.hpp"
#include "bhns.hpp"
#include "Configurator/config_binary.hpp"

using namespace Kadath;
using namespace Kadath::FUKA_Config;

#include "fuka_exporter.h"

BinaryInfo read_binary_info(BinaryType binary_type, char *info_filename)
{
  kadath_config_boost<BIN_INFO> bconfig(info_filename);
  BinaryInfo binary_info;
  double com = bconfig(COM);

  switch (binary_type)
  {
  case BNS:
  {
    binary_info.mass1 = bconfig(MADM, BCO1);
    binary_info.mass2 = bconfig(MADM, BCO2);

    std::string space_filename = bconfig.space_filename();
    FILE *space_file = fopen(space_filename.c_str(), "r");

    Space_bin_ns space(space_file);
    Index center_pos(space.get_domain(space.NS1)->get_nbr_points());
    binary_info.position_x1 = com + space.get_domain(space.NS1)->get_cart(1)(center_pos);
    binary_info.position_x2 = com + space.get_domain(space.NS2)->get_cart(1)(center_pos);
    break;
  }
  case BBH:
  {
    binary_info.mass1 = bconfig(MCH, BCO1);
    binary_info.mass2 = bconfig(MCH, BCO2);

    std::string space_filename = bconfig.space_filename();
    FILE *space_file = fopen(space_filename.c_str(), "r");

    Space_bin_bh space(space_file);
    Index center_pos(space.get_domain(space.BH1)->get_nbr_points());
    binary_info.position_x1 = com + space.get_domain(space.BH1)->get_cart(1)(center_pos);
    binary_info.position_x2 = com + space.get_domain(space.BH2)->get_cart(1)(center_pos);
    break;
  }
  case BHNS:
  {
    binary_info.mass1 = bconfig(MADM, BCO1);
    binary_info.mass2 = bconfig(MCH, BCO2);

    std::string space_filename = bconfig.space_filename();
    FILE *space_file = fopen(space_filename.c_str(), "r");

    Space_bhns space(space_file);
    Index center_pos(space.get_domain(space.NS)->get_nbr_points());
    binary_info.position_x1 = com + space.get_domain(space.NS)->get_cart(1)(center_pos);
    binary_info.position_x2 = com + space.get_domain(space.BH)->get_cart(1)(center_pos);
    break;
  }
  }

  return binary_info;
}

void copy_vector_to_array(double *arr, std::vector<double> v)
{
  std::copy(v.begin(), v.end(), arr);
}

double *allocate_double(int n)
{
  double *pointer = (double *)calloc(n, sizeof(double));
  if (pointer == NULL)
  {
    printf("cannot allocate memory\n");
    exit(1);
  }
  return pointer;
}

Fields allocate_fields(int n_points)
{
  Fields fields = {
    alpha : allocate_double(n_points),
    beta_x : allocate_double(n_points),
    beta_y : allocate_double(n_points),
    beta_z : allocate_double(n_points),
    gamma_xx : allocate_double(n_points),
    gamma_xy : allocate_double(n_points),
    gamma_xz : allocate_double(n_points),
    gamma_yy : allocate_double(n_points),
    gamma_yz : allocate_double(n_points),
    gamma_zz : allocate_double(n_points),
    K_xx : allocate_double(n_points),
    K_xy : allocate_double(n_points),
    K_xz : allocate_double(n_points),
    K_yy : allocate_double(n_points),
    K_yz : allocate_double(n_points),
    K_zz : allocate_double(n_points),
    // Hydro
    rho : allocate_double(n_points),
    epsilon : allocate_double(n_points),
    pressure : allocate_double(n_points),
    v_x : allocate_double(n_points),
    v_y : allocate_double(n_points),
    v_z : allocate_double(n_points),
  };
  return fields;
}

void free_fields(Fields *fields)
{
  free(fields->alpha);
  free(fields->beta_x);

  free(fields->beta_y);

  free(fields->beta_z);

  free(fields->gamma_xx);
  free(fields->gamma_xy);
  free(fields->gamma_xz);
  free(fields->gamma_yy);
  free(fields->gamma_yz);
  free(fields->gamma_zz);

  free(fields->K_xx);
  free(fields->K_xy);
  free(fields->K_xz);
  free(fields->K_yy);
  free(fields->K_yz);
  free(fields->K_zz);

  free(fields->rho);
  free(fields->epsilon);
  free(fields->pressure);
  free(fields->v_x);
  free(fields->v_y);
  free(fields->v_z);
}

void copy_fields(Fields *dst_fields, Fields *src_fields, int offset, int size)
{
  memcpy(dst_fields->alpha + offset, src_fields->alpha, size);

  memcpy(dst_fields->beta_x + offset, src_fields->beta_x, size);
  memcpy(dst_fields->beta_y + offset, src_fields->beta_y, size);
  memcpy(dst_fields->beta_z + offset, src_fields->beta_z, size);

  memcpy(dst_fields->gamma_xx + offset, src_fields->gamma_xx, size);
  memcpy(dst_fields->gamma_xy + offset, src_fields->gamma_xy, size);
  memcpy(dst_fields->gamma_xz + offset, src_fields->gamma_xz, size);
  memcpy(dst_fields->gamma_yy + offset, src_fields->gamma_yy, size);
  memcpy(dst_fields->gamma_yz + offset, src_fields->gamma_yz, size);
  memcpy(dst_fields->gamma_zz + offset, src_fields->gamma_zz, size);

  memcpy(dst_fields->K_xx + offset, src_fields->K_xx, size);
  memcpy(dst_fields->K_xy + offset, src_fields->K_xy, size);
  memcpy(dst_fields->K_xz + offset, src_fields->K_xz, size);
  memcpy(dst_fields->K_yy + offset, src_fields->K_yy, size);
  memcpy(dst_fields->K_yz + offset, src_fields->K_yz, size);
  memcpy(dst_fields->K_zz + offset, src_fields->K_zz, size);


  memcpy(dst_fields->rho + offset, src_fields->rho, size);
  memcpy(dst_fields->epsilon + offset, src_fields->epsilon, size);
  memcpy(dst_fields->pressure + offset, src_fields->pressure, size);
  memcpy(dst_fields->v_x + offset, src_fields->v_x, size);
  memcpy(dst_fields->v_y + offset, src_fields->v_y, size);
  memcpy(dst_fields->v_z + offset, src_fields->v_z, size);
}

Fields
__interpolate_FUKA_ID(FUKAInterpolateRequest *req)
{
  Grid *grid = req->grid;
  // In the case with no matter, the exported array is shorter,
  // so we have to take care of this.
  std::array<std::vector<double>, N_VARIABLES_VACUUM> exported_vacuum;
  std::array<std::vector<double>, N_VARIABLES_MATTER> exported_matter;

  switch (req->binary_type)
  {
  case BNS:
  {
    exported_matter = KadathExportBNS(grid->n_points, grid->x, grid->y, grid->z, req->info_filename);
    break;
  }
  case BBH:
  {
    exported_vacuum = KadathExportBBH(grid->n_points, grid->x, grid->y, grid->z, req->info_filename,
                                      req->interpolation_offset, req->interpolation_order, req->relative_dr_spacing);
    break;
  }
  case BHNS:
  {
    exported_matter = KadathExportBHNS(grid->n_points, grid->x, grid->y, grid->z, req->info_filename,
                                       req->interpolation_offset, req->interpolation_order, req->relative_dr_spacing);
    break;
  }
  default:
  {
    std::cout << "Unknown binary type" << std::endl;
    return Fields{};
  }
  }

  Fields fields = allocate_fields(req->grid->n_points);

  // Copy is needed as the underlying array is owned
  // by std::vector on stack, so it will be deallocated as
  // soon as this function returns.

  if (req->binary_type == BBH)
  {
    copy_vector_to_array(fields.alpha, exported_vacuum[ALPHA]);

    copy_vector_to_array(fields.beta_x, exported_vacuum[BETAX]);
    copy_vector_to_array(fields.beta_y, exported_vacuum[BETAY]);
    copy_vector_to_array(fields.beta_z, exported_vacuum[BETAZ]);

    copy_vector_to_array(fields.gamma_xx, exported_vacuum[GXX]);
    copy_vector_to_array(fields.gamma_xy, exported_vacuum[GXY]);
    copy_vector_to_array(fields.gamma_xz, exported_vacuum[GXZ]);
    copy_vector_to_array(fields.gamma_yy, exported_vacuum[GYY]);
    copy_vector_to_array(fields.gamma_yz, exported_vacuum[GYZ]);
    copy_vector_to_array(fields.gamma_zz, exported_vacuum[GZZ]);

    copy_vector_to_array(fields.K_xx, exported_vacuum[KXX]);
    copy_vector_to_array(fields.K_xy, exported_vacuum[KXY]);
    copy_vector_to_array(fields.K_xz, exported_vacuum[KXZ]);
    copy_vector_to_array(fields.K_yy, exported_vacuum[KYY]);
    copy_vector_to_array(fields.K_yz, exported_vacuum[KYZ]);
    copy_vector_to_array(fields.K_zz, exported_vacuum[KZZ]);
  }

  if (req->binary_type == BNS || req->binary_type == BHNS)
  {
    copy_vector_to_array(fields.alpha, exported_matter[ALPHA]);

    copy_vector_to_array(fields.beta_x, exported_matter[BETAX]);
    copy_vector_to_array(fields.beta_y, exported_matter[BETAY]);
    copy_vector_to_array(fields.beta_z, exported_matter[BETAZ]);

    copy_vector_to_array(fields.gamma_xx, exported_matter[GXX]);
    copy_vector_to_array(fields.gamma_xy, exported_matter[GXY]);
    copy_vector_to_array(fields.gamma_xz, exported_matter[GXZ]);
    copy_vector_to_array(fields.gamma_yy, exported_matter[GYY]);
    copy_vector_to_array(fields.gamma_yz, exported_matter[GYZ]);
    copy_vector_to_array(fields.gamma_zz, exported_matter[GZZ]);

    copy_vector_to_array(fields.K_xx, exported_matter[KXX]);
    copy_vector_to_array(fields.K_xy, exported_matter[KXY]);
    copy_vector_to_array(fields.K_xz, exported_matter[KXZ]);
    copy_vector_to_array(fields.K_yy, exported_matter[KYY]);
    copy_vector_to_array(fields.K_yz, exported_matter[KYZ]);
    copy_vector_to_array(fields.K_zz, exported_matter[KZZ]);

    copy_vector_to_array(fields.rho, exported_matter[RHO]);
    copy_vector_to_array(fields.epsilon, exported_matter[EPS]);
    copy_vector_to_array(fields.pressure, exported_matter[PRESS]);
    copy_vector_to_array(fields.v_x, exported_matter[VELX]);
    copy_vector_to_array(fields.v_y, exported_matter[VELY]);
    copy_vector_to_array(fields.v_z, exported_matter[VELZ]);
  }

  return fields;
}

Fields
interpolate_FUKA_ID(FUKAInterpolateRequest *req)
{
  // omp_set_num_threads(4);
  int n_chunks = omp_get_max_threads();
  Grid *grid = req->grid;

  Fields fields = allocate_fields(grid->n_points);

  #pragma omp parallel for
  for (int chunk = 0; chunk < n_chunks; chunk++)
  {
    int chunk_size = grid->n_points / n_chunks;
    int offset = chunk * chunk_size;
    if (chunk == n_chunks - 1)
    {
      chunk_size += grid->n_points % n_chunks;
    }
    if (chunk != 1 && chunk != 2) {
      continue;
    }
    #pragma omp critical
    std::cout << "chunk: " << chunk << "/" << n_chunks << ", size: " << chunk_size << ", offset: " << offset << std::endl;
    // Copy request
    FUKAInterpolateRequest chunk_request = *req;
    Grid chunk_grid = {
      x : &(grid->x[offset]),
      y : &(grid->y[offset]),
      z : &(grid->z[offset]),
      n_points : chunk_size,
    };

    chunk_request.grid = &chunk_grid;
    Fields chunk_fields = __interpolate_FUKA_ID(&chunk_request);
    #pragma omp critical
    copy_fields(&fields, &chunk_fields, offset, chunk_size);
  }
  return fields;
}